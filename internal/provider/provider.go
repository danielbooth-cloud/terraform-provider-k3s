// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ provider.Provider = &k3sProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &k3sProvider{
			version: version,
		}
	}
}

type k3sProvider struct {
	version string
}

// k3sProviderModel maps provder schema data to Go type
type k3sProviderModel struct {
	// K3s version to select, if not selected
	// will default to latest
	Version types.String `tfsdk:"k3s_version"`
}

// Metadata returns the provider type name.
func (p *k3sProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "k3s"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *k3sProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"k3s_version": schema.StringAttribute{
				Optional:    true,
				Description: "K3s version to select, if not selected will default to latest",
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *k3sProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config k3sProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.Version.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown K3s Version",
			"The provider cannot create the K3s API client as there is an unknown configuration value for the K3s Version. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the K3S_VERSION environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	version := os.Getenv("K3S_VERSION")

	if !config.Version.IsNull() {
		version = config.Version.ValueString()
	}

	if version == "" || strings.ToLower(version) == "latest" {
		version, err := getLatestRelease()
		if err != nil {
			resp.Diagnostics.Append(fromError("Retrieving k3s releases from Github", err))
		}
		p.version = version
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *k3sProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *k3sProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewK3sServerResource,
	}
}

type versionDiagnositcs struct {
	severity diag.Severity
	summary  string
	detail   string
}

func fromError(summary string, e error) diag.Diagnostic {
	return versionDiagnositcs{
		severity: 1,
		summary:  summary,
		detail:   e.Error(),
	}
}

func (v versionDiagnositcs) Severity() diag.Severity {
	return v.severity
}
func (v versionDiagnositcs) Summary() string {
	return v.summary
}
func (v versionDiagnositcs) Detail() string {
	return v.detail
}
func (v versionDiagnositcs) Equal(o diag.Diagnostic) bool {
	return v.severity == o.Severity() && v.summary == o.Summary() && v.detail == o.Detail()
}

func getLatestRelease() (string, error) {
	// Get releases
	resp, err := http.Get("https://api.github.com/repos/k3s-io/k3s/releases")
	if err != nil {
		return "", err
	}

	// Read body
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Json into Dicts
	var releases []struct {
		Tag string `json:"tag_name"`
	}
	err = json.Unmarshal(body, &releases)
	if err != nil {
		return "", err
	}

	return releases[0].Tag, nil
}
