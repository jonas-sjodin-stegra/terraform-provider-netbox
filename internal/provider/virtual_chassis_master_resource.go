package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fbreckle/go-netbox/netbox/client/dcim"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/e-breuninger/terraform-provider-netbox/internal/netboxapi"
)

// netbox_virtual_chassis_master breaks the dependency cycle between a chassis and its member.

func init() {
	companionResources = append(companionResources, NewVirtualChassisMasterResource)
}

var (
	_ resource.Resource                = (*virtualChassisMasterResource)(nil)
	_ resource.ResourceWithConfigure   = (*virtualChassisMasterResource)(nil)
	_ resource.ResourceWithImportState = (*virtualChassisMasterResource)(nil)
)

func NewVirtualChassisMasterResource() resource.Resource {
	return &virtualChassisMasterResource{}
}

type virtualChassisMasterResource struct {
	client *netboxapi.Client
}

type virtualChassisMasterResourceModel struct {
	ID               types.String `tfsdk:"id"`
	VirtualChassisID types.Int64  `tfsdk:"virtual_chassis_id"`
	DeviceID         types.Int64  `tfsdk:"device_id"`
}

func (resource *virtualChassisMasterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_chassis_master"
}

func (resource *virtualChassisMasterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: ":meta:subcategory:Data Center Inventory Management (DCIM):Assigns a member device as the master of a virtual chassis. Deleting this resource clears the assignment. Import with the virtual chassis id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The virtual chassis id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"virtual_chassis_id": schema.Int64Attribute{
				Required:      true,
				Description:   "Id of the virtual chassis.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"device_id": schema.Int64Attribute{
				Required:    true,
				Description: "Id of the member device to designate as master.",
			},
		},
	}
}

func (resource *virtualChassisMasterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*netboxapi.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource configure type",
			fmt.Sprintf("Expected *netboxapi.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	resource.client = client
}

func (resource *virtualChassisMasterResource) set(ctx context.Context, plan *virtualChassisMasterResourceModel) error {
	deviceID := plan.DeviceID.ValueInt64()
	if err := patchVirtualChassisMaster(ctx, resource.client, plan.VirtualChassisID.ValueInt64(), deviceID); err != nil {
		return err
	}
	plan.ID = types.StringValue(strconv.FormatInt(plan.VirtualChassisID.ValueInt64(), 10))
	return nil
}

func (resource *virtualChassisMasterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan virtualChassisMasterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := resource.set(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting the virtual chassis master", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (resource *virtualChassisMasterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state virtualChassisMasterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid netbox_virtual_chassis_master id", fmt.Sprintf("Expected the virtual chassis id, got %q.", state.ID.ValueString()))
		return
	}
	res, err := resource.client.Dcim.DcimVirtualChassisRetrieveContext(ctx, dcim.NewDcimVirtualChassisRetrieveParams().WithID(id), nil)
	if netboxapi.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading the virtual chassis", err.Error())
		return
	}
	if res.Payload.Master == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.VirtualChassisID = types.Int64Value(id)
	state.DeviceID = types.Int64Value(res.Payload.Master.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (resource *virtualChassisMasterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan virtualChassisMasterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := resource.set(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting the virtual chassis master", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (resource *virtualChassisMasterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state virtualChassisMasterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := patchVirtualChassisMaster(ctx, resource.client, state.VirtualChassisID.ValueInt64(), nil)
	if err != nil && !netboxapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Error clearing the virtual chassis master", err.Error())
	}
}

func (resource *virtualChassisMasterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected the virtual chassis id, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &virtualChassisMasterResourceModel{
		ID: types.StringValue(req.ID), VirtualChassisID: types.Int64Value(id), DeviceID: types.Int64Null(),
	})...)
}

func patchVirtualChassisMaster(ctx context.Context, client *netboxapi.Client, id int64, master any) error {
	params := dcim.NewDcimVirtualChassisPartialUpdateParams().WithID(id)
	_, err := client.Dcim.DcimVirtualChassisPartialUpdateContext(ctx, params, nil, netboxapi.WithBody(map[string]any{"master": master}))
	return err
}
