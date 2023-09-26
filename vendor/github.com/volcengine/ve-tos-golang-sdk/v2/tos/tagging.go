package tos

import (
	"context"
)

func (cli *ClientV2) PutObjectTagging(ctx context.Context, input *PutObjectTaggingInput) (*PutObjectTaggingOutput, error) {
	return cli.baseClient.PutObjectTagging(ctx, input)
}

func (cli *ClientV2) GetObjectTagging(ctx context.Context, input *GetObjectTaggingInput) (*GetObjectTaggingOutput, error) {
	return cli.baseClient.GetObjectTagging(ctx, input)
}

func (cli *ClientV2) DeleteObjectTagging(ctx context.Context, input *DeleteObjectTaggingInput) (*DeleteObjectTaggingOutput, error) {
	return cli.baseClient.DeleteObjectTagging(ctx, input)
}
