package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/plugin"
	"github.com/turbot/steampipe-plugin-sdk/plugin/transform"
)

func tableAwsEc2TransitGatewayVpcAttachment(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name: "aws_ec2_transit_gateway_vpc_attachment",
		Get: &plugin.GetConfig{
			KeyColumns:        plugin.SingleColumn("transit_gateway_attachment_id"),
			ShouldIgnoreError: isNotFoundError([]string{"InvalidTransitGatewayAttachmentID.NotFound", "InvalidTransitGatewayAttachmentID.Unavailable", "InvalidTransitGatewayAttachmentID.Malformed"}),
			Hydrate:           getEc2TransitGatewayVpcAttachment,
		},
		List: &plugin.ListConfig{
			Hydrate: listEc2TransitGatewayVpcAttachment,
		},
		GetMatrixItem: BuildRegionList,
		Columns: awsRegionalColumns([]*plugin.Column{
			{
				Name:        "transit_gateway_attachment_id",
				Description: "The ID of the transit gateway attachment.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "transit_gateway_id",
				Description: "The ID of the transit gateway.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "transit_gateway_owner_id",
				Description: "The ID of the AWS account that owns the transit gateway.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "state",
				Description: "The attachment state of the transit gateway attachment.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "creation_time",
				Description: "The creation time of the transit gateway attachment.",
				Type:        proto.ColumnType_TIMESTAMP,
			},
			{
				Name:        "resource_id",
				Description: "The ID of the resource.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "resource_type",
				Description: "The resource type of the transit gateway attachment.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "resource_owner_id",
				Description: "The ID of the AWS account that owns the resource.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "association_state",
				Description: "The state of the association.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Association.State"),
			},
			{
				Name:        "association_transit_gateway_route_table_id",
				Description: "The ID of the route table for the transit gateway.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Association.TransitGatewayRouteTableId"),
			},
			{
				Name:        "tags_src",
				Description: "A list of tags assigned.",
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("Tags"),
			},

			/// Standard columns
			{
				Name:        "tags",
				Description: resourceInterfaceDescription("tags"),
				Type:        proto.ColumnType_JSON,
				Transform:   transform.From(transitGatewayAttachmentRawTagsToTurbotTags),
			},
			{
				Name:        "title",
				Description: resourceInterfaceDescription("title"),
				Type:        proto.ColumnType_STRING,
				Transform:   transform.From(getEc2TransitGatewayAttachmentTitle),
			},
			{
				Name:        "akas",
				Description: resourceInterfaceDescription("akas"),
				Type:        proto.ColumnType_JSON,
				Hydrate:     getAwsEc2TransitGatewayVpcAttachmentAkas,
				Transform:   transform.FromValue(),
			},
		}),
	}
}

//// LIST FUNCTION

func listEc2TransitGatewayVpcAttachment(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := d.KeyColumnQualString(matrixKeyRegion)
	plugin.Logger(ctx).Trace("listEc2TransitGatewayVpcAttachment", "AWS_REGION", region)

	// Create Session
	svc, err := Ec2Service(ctx, d, region)
	if err != nil {
		return nil, err
	}

	// List call
	err = svc.DescribeTransitGatewayAttachmentsPages(
		&ec2.DescribeTransitGatewayAttachmentsInput{},
		func(page *ec2.DescribeTransitGatewayAttachmentsOutput, isLast bool) bool {
			for _, transitGatewayAttachment := range page.TransitGatewayAttachments {
				d.StreamListItem(ctx, transitGatewayAttachment)
			}
			return !isLast
		},
	)

	return nil, err
}

//// HYDRATE FUNCTIONS

func getEc2TransitGatewayVpcAttachment(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getEc2TransitGatewayVpcAttachment")

	region := d.KeyColumnQualString(matrixKeyRegion)
	transitGatewayAttachmentID := d.KeyColumnQuals["transit_gateway_attachment_id"].GetStringValue()

	// Create Session
	svc, err := Ec2Service(ctx, d, region)
	if err != nil {
		return nil, err
	}

	// Build params
	params := &ec2.DescribeTransitGatewayAttachmentsInput{
		TransitGatewayAttachmentIds: []*string{aws.String(transitGatewayAttachmentID)},
	}

	op, err := svc.DescribeTransitGatewayAttachments(params)
	if err != nil {
		plugin.Logger(ctx).Debug("getEc2TransitGatewayVpcAttachment__", "ERROR", err)
		return nil, err
	}

	if op.TransitGatewayAttachments != nil && len(op.TransitGatewayAttachments) > 0 {
		return op.TransitGatewayAttachments[0], nil
	}
	return nil, nil
}

func getAwsEc2TransitGatewayVpcAttachmentAkas(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getAwsEc2TransitGatewayVpcAttachmentAkas")
	region := d.KeyColumnQualString(matrixKeyRegion)
	transitGatewayAttachment := h.Item.(*ec2.TransitGatewayAttachment)

	getCommonColumnsCached := plugin.HydrateFunc(getCommonColumns).WithCache()
	commonData, err := getCommonColumnsCached(ctx, d, h)
	if err != nil {
		return nil, err
	}
	commonColumnData := commonData.(*awsCommonColumnData)

	// Get the resource akas
	akas := []string{"arn:" + commonColumnData.Partition + ":ec2:" + region + ":" + commonColumnData.AccountId + ":transit-gateway-attachment/" + *transitGatewayAttachment.TransitGatewayAttachmentId}

	return akas, nil
}

//// TRANSFORM FUNCTIONS

func transitGatewayAttachmentRawTagsToTurbotTags(_ context.Context, d *transform.TransformData) (interface{}, error) {
	data := d.HydrateItem.(*ec2.TransitGatewayAttachment)
	return ec2TagsToMap(data.Tags)
}

func getEc2TransitGatewayAttachmentTitle(_ context.Context, d *transform.TransformData) (interface{}, error) {
	data := d.HydrateItem.(*ec2.TransitGatewayAttachment)
	title := data.TransitGatewayAttachmentId
	if data.Tags != nil {
		for _, i := range data.Tags {
			if *i.Key == "Name" {
				title = i.Value
			}
		}
	}
	return title, nil
}
