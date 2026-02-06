// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compute

type SSubImage struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	MinDiskMB  int32  `json:"min_disk_mb"`
	DiskFormat string `json:"disk_format"`
}

type SImagesInGuest struct {
	Id         string      `json:"id"`
	Name       string      `json:"name"`
	RootImage  SSubImage   `json:"root_image"`
	DataImages []SSubImage `json:"data_images"`
}

type SGuestScreenDumpInfo struct {
	S3AccessKey  string `json:"s3_access_key"`
	S3SecretKey  string `json:"s3_secret_key"`
	S3Endpoint   string `json:"s3_endpoint"`
	S3BucketName string `json:"s3_bucket_name"`
	S3ObjectName string `json:"s3_object_name"`
	S3UseSSL     bool   `json:"s3_use_ssl"`
}

type GuestScreenDumpListInput struct {
	Server string `json:"server"`
}

type GetDetailsGuestScreenDumpInput struct {
	ObjectName string `json:"object_name"`
}

type GetDetailsGuestScreenDumpOutput struct {
	GuestId    string `json:"guest_id"`
	Name       string `json:"name"`
	ScreenDump string `json:"screen_dump"`
}
