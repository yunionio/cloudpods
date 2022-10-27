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

package aws

type SDBInstanceSpec struct {
	VcpuCount  int
	VmemSizeMb int
}

var DBInstanceSpecs = map[string]SDBInstanceSpec{
	"db.m1.small":  {VcpuCount: 1, VmemSizeMb: 1.75 * 1024.0},
	"db.m1.medium": {VcpuCount: 1, VmemSizeMb: 3.75 * 1024.0},
	"db.m1.large":  {VcpuCount: 2, VmemSizeMb: 7.5 * 1024},
	"db.m1.xlarge": {VcpuCount: 4, VmemSizeMb: 15 * 1024},

	"db.t3.micro":   {VcpuCount: 2, VmemSizeMb: 1 * 1024},
	"db.t3.small":   {VcpuCount: 2, VmemSizeMb: 2 * 1024},
	"db.t3.medium":  {VcpuCount: 2, VmemSizeMb: 4 * 1024},
	"db.t3.large":   {VcpuCount: 2, VmemSizeMb: 8 * 1024},
	"db.t3.xlarge":  {VcpuCount: 4, VmemSizeMb: 16 * 1024},
	"db.t3.2xlarge": {VcpuCount: 8, VmemSizeMb: 32 * 1024},

	"db.t2.micro":   {VcpuCount: 1, VmemSizeMb: 1 * 1024},
	"db.t2.small":   {VcpuCount: 1, VmemSizeMb: 1 * 1024},
	"db.t2.medium":  {VcpuCount: 2, VmemSizeMb: 2 * 1024},
	"db.t2.large":   {VcpuCount: 2, VmemSizeMb: 2 * 1024},
	"db.t2.xlarge":  {VcpuCount: 4, VmemSizeMb: 4 * 1024},
	"db.t2.2xlarge": {VcpuCount: 8, VmemSizeMb: 8 * 1024},

	"db.t4g.micro":   {VcpuCount: 2, VmemSizeMb: 1 * 1024},
	"db.t4g.small":   {VcpuCount: 2, VmemSizeMb: 2 * 1024},
	"db.t4g.medium":  {VcpuCount: 2, VmemSizeMb: 4 * 1024},
	"db.t4g.large":   {VcpuCount: 2, VmemSizeMb: 8 * 1024},
	"db.t4g.xlarge":  {VcpuCount: 4, VmemSizeMb: 16 * 1024},
	"db.t4g.2xlarge": {VcpuCount: 8, VmemSizeMb: 32 * 1024},

	"db.m6g.large":    {VcpuCount: 2, VmemSizeMb: 8 * 1024},
	"db.m6g.xlarge":   {VcpuCount: 4, VmemSizeMb: 16 * 1024},
	"db.m6g.2xlarge":  {VcpuCount: 8, VmemSizeMb: 32 * 1024},
	"db.m6g.4xlarge":  {VcpuCount: 16, VmemSizeMb: 64 * 1024},
	"db.m6g.8xlarge":  {VcpuCount: 32, VmemSizeMb: 128 * 1024},
	"db.m6g.12xlarge": {VcpuCount: 48, VmemSizeMb: 192 * 1024},
	"db.m6g.16xlarge": {VcpuCount: 64, VmemSizeMb: 256 * 1024},

	"db.m5.large":    {VcpuCount: 2, VmemSizeMb: 8 * 1024},
	"db.m5.xlarge":   {VcpuCount: 4, VmemSizeMb: 16 * 1024},
	"db.m5.2xlarge":  {VcpuCount: 8, VmemSizeMb: 32 * 1024},
	"db.m5.4xlarge":  {VcpuCount: 16, VmemSizeMb: 64 * 1024},
	"db.m5.8xlarge":  {VcpuCount: 32, VmemSizeMb: 128 * 1024},
	"db.m5.12xlarge": {VcpuCount: 48, VmemSizeMb: 192 * 1024},
	"db.m5.16xlarge": {VcpuCount: 64, VmemSizeMb: 256 * 1024},
	"db.m5.24xlarge": {VcpuCount: 96, VmemSizeMb: 384 * 1024},

	"db.m5d.large":    {VcpuCount: 2, VmemSizeMb: 8 * 1024},
	"db.m5d.xlarge":   {VcpuCount: 4, VmemSizeMb: 16 * 1024},
	"db.m5d.2xlarge":  {VcpuCount: 8, VmemSizeMb: 32 * 1024},
	"db.m5d.4xlarge":  {VcpuCount: 16, VmemSizeMb: 64 * 1024},
	"db.m5d.8xlarge":  {VcpuCount: 32, VmemSizeMb: 128 * 1024},
	"db.m5d.12xlarge": {VcpuCount: 48, VmemSizeMb: 192 * 1024},
	"db.m5d.16xlarge": {VcpuCount: 64, VmemSizeMb: 256 * 1024},
	"db.m5d.24xlarge": {VcpuCount: 96, VmemSizeMb: 384 * 1024},

	"db.r6g.large":    {VcpuCount: 2, VmemSizeMb: 16 * 1024},
	"db.r6g.xlarge":   {VcpuCount: 4, VmemSizeMb: 32 * 1024},
	"db.r6g.2xlarge":  {VcpuCount: 8, VmemSizeMb: 64 * 1024},
	"db.r6g.4xlarge":  {VcpuCount: 16, VmemSizeMb: 128 * 1024},
	"db.r6g.8xlarge":  {VcpuCount: 32, VmemSizeMb: 256 * 1024},
	"db.r6g.12xlarge": {VcpuCount: 48, VmemSizeMb: 384 * 1024},
	"db.r6g.16xlarge": {VcpuCount: 64, VmemSizeMb: 512 * 1024},

	"db.r6gd.medium":   {VcpuCount: 1, VmemSizeMb: 8 * 1024},
	"db.r6gd.large":    {VcpuCount: 2, VmemSizeMb: 16 * 1024},
	"db.r6gd.xlarge":   {VcpuCount: 4, VmemSizeMb: 32 * 1024},
	"db.r6gd.2xlarge":  {VcpuCount: 8, VmemSizeMb: 64 * 1024},
	"db.r6gd.4xlarge":  {VcpuCount: 16, VmemSizeMb: 128 * 1024},
	"db.r6gd.8xlarge":  {VcpuCount: 32, VmemSizeMb: 256 * 1024},
	"db.r6gd.12xlarge": {VcpuCount: 48, VmemSizeMb: 383 * 1024},
	"db.r6gd.16xlarge": {VcpuCount: 64, VmemSizeMb: 512 * 1024},
	"db.r6gd.metal":    {VcpuCount: 64, VmemSizeMb: 512 * 1024},

	"db.r5.large":    {VcpuCount: 2, VmemSizeMb: 16 * 1024},
	"db.r5.xlarge":   {VcpuCount: 4, VmemSizeMb: 32 * 1024},
	"db.r5.2xlarge":  {VcpuCount: 8, VmemSizeMb: 64 * 1024},
	"db.r5.4xlarge":  {VcpuCount: 16, VmemSizeMb: 128 * 1024},
	"db.r5.8xlarge":  {VcpuCount: 32, VmemSizeMb: 256 * 1024},
	"db.r5.12xlarge": {VcpuCount: 48, VmemSizeMb: 384 * 1024},
	"db.r5.16xlarge": {VcpuCount: 64, VmemSizeMb: 512 * 1024},
	"db.r5.24xlarge": {VcpuCount: 96, VmemSizeMb: 768 * 1024},

	"db.r5b.large":    {VcpuCount: 2, VmemSizeMb: 16 * 1024},
	"db.r5b.xlarge":   {VcpuCount: 4, VmemSizeMb: 32 * 1024},
	"db.r5b.2xlarge":  {VcpuCount: 8, VmemSizeMb: 64 * 1024},
	"db.r5b.4xlarge":  {VcpuCount: 16, VmemSizeMb: 128 * 1024},
	"db.r5b.8xlarge":  {VcpuCount: 32, VmemSizeMb: 256 * 1024},
	"db.r5b.12xlarge": {VcpuCount: 48, VmemSizeMb: 384 * 1024},
	"db.r5b.16xlarge": {VcpuCount: 64, VmemSizeMb: 512 * 1024},
	"db.r5b.24xlarge": {VcpuCount: 96, VmemSizeMb: 768 * 1024},

	"db.r5d.large":    {VcpuCount: 2, VmemSizeMb: 16 * 1024},
	"db.r5d.xlarge":   {VcpuCount: 4, VmemSizeMb: 32 * 1024},
	"db.r5d.2xlarge":  {VcpuCount: 8, VmemSizeMb: 64 * 1024},
	"db.r5d.4xlarge":  {VcpuCount: 16, VmemSizeMb: 128 * 1024},
	"db.r5d.8xlarge":  {VcpuCount: 32, VmemSizeMb: 256 * 1024},
	"db.r5d.12xlarge": {VcpuCount: 48, VmemSizeMb: 384 * 1024},
	"db.r5d.16xlarge": {VcpuCount: 64, VmemSizeMb: 512 * 1024},
	"db.r5d.24xlarge": {VcpuCount: 96, VmemSizeMb: 768 * 1024},

	"db.x1e.xlarge":   {VcpuCount: 4, VmemSizeMb: 122 * 1024},
	"db.x1e.2xlarge":  {VcpuCount: 8, VmemSizeMb: 244 * 1024},
	"db.x1e.4xlarge":  {VcpuCount: 16, VmemSizeMb: 488 * 1024},
	"db.x1e.8xlarge":  {VcpuCount: 32, VmemSizeMb: 976 * 1024},
	"db.x1e.16xlarge": {VcpuCount: 64, VmemSizeMb: 1952 * 1024},
	"db.x1e.32xlarge": {VcpuCount: 128, VmemSizeMb: 3904 * 1024},

	"db.x1.16xlarge": {VcpuCount: 64, VmemSizeMb: 976 * 1024},
	"db.x1.32xlarge": {VcpuCount: 128, VmemSizeMb: 1952 * 1024},

	"db.z1d.large":    {VcpuCount: 2, VmemSizeMb: 16 * 1024},
	"db.z1d.xlarge":   {VcpuCount: 4, VmemSizeMb: 32 * 1024},
	"db.z1d.2xlarge":  {VcpuCount: 8, VmemSizeMb: 64 * 1024},
	"db.z1d.3xlarge":  {VcpuCount: 12, VmemSizeMb: 96 * 1024},
	"db.z1d.6xlarge":  {VcpuCount: 24, VmemSizeMb: 192 * 1024},
	"db.z1d.12xlarge": {VcpuCount: 48, VmemSizeMb: 384 * 1024},

	"db.r5.large.tpc1.mem2x":    {VcpuCount: 2, VmemSizeMb: 32 * 1024},
	"db.r5.xlarge.tpc2.mem2x":   {VcpuCount: 4, VmemSizeMb: 64 * 1024},
	"db.r5.xlarge.tpc2.mem4x":   {VcpuCount: 4, VmemSizeMb: 128 * 1024},
	"db.r5.2xlarge.tpc1.mem2x":  {VcpuCount: 8, VmemSizeMb: 128 * 1024},
	"db.r5.2xlarge.tpc2.mem4x":  {VcpuCount: 8, VmemSizeMb: 256 * 1024},
	"db.r5.2xlarge.tpc2.mem8x":  {VcpuCount: 8, VmemSizeMb: 512 * 1024},
	"db.r5.4xlarge.tpc2.mem2x":  {VcpuCount: 16, VmemSizeMb: 256 * 1024},
	"db.r5.4xlarge.tpc2.mem3x":  {VcpuCount: 16, VmemSizeMb: 384 * 1024},
	"db.r5.4xlarge.tpc2.mem4x":  {VcpuCount: 16, VmemSizeMb: 512 * 1024},
	"db.r5.6xlarge.tpc2.mem4x":  {VcpuCount: 24, VmemSizeMb: 768 * 1024},
	"db.r5.8xlarge.tpc2.mem3x":  {VcpuCount: 32, VmemSizeMb: 768 * 1024},
	"db.r5.12xlarge.tpc2.mem2x": {VcpuCount: 48, VmemSizeMb: 768 * 1024},

	"db.m4.large":    {VcpuCount: 2, VmemSizeMb: 8 * 1024},
	"db.m4.xlarge":   {VcpuCount: 4, VmemSizeMb: 16 * 1024},
	"db.m4.2xlarge":  {VcpuCount: 8, VmemSizeMb: 32 * 1024},
	"db.m4.4xlarge":  {VcpuCount: 16, VmemSizeMb: 64 * 1024},
	"db.m4.10xlarge": {VcpuCount: 40, VmemSizeMb: 160 * 1024},
	"db.m4.16xlarge": {VcpuCount: 64, VmemSizeMb: 256 * 1024},

	"db.m3.medium":  {VcpuCount: 1, VmemSizeMb: 3.75 * 1024},
	"db.m3.large":   {VcpuCount: 2, VmemSizeMb: 7.5 * 1024},
	"db.m3.xlarge":  {VcpuCount: 4, VmemSizeMb: 15 * 1024},
	"db.m3.2xlarge": {VcpuCount: 8, VmemSizeMb: 30 * 1024},

	"db.r3.large":   {VcpuCount: 2, VmemSizeMb: 15.25 * 1024},
	"db.r3.xlarge":  {VcpuCount: 4, VmemSizeMb: 30.5 * 1024},
	"db.r3.2xlarge": {VcpuCount: 8, VmemSizeMb: 61 * 1024},
	"db.r3.4xlarge": {VcpuCount: 16, VmemSizeMb: 122 * 1024},
	"db.r3.8xlarge": {VcpuCount: 32, VmemSizeMb: 244 * 1024},

	"db.r4.large":    {VcpuCount: 2, VmemSizeMb: 15.25 * 1024},
	"db.r4.xlarge":   {VcpuCount: 4, VmemSizeMb: 30.5 * 1024},
	"db.r4.2xlarge":  {VcpuCount: 8, VmemSizeMb: 61 * 1024},
	"db.r4.4xlarge":  {VcpuCount: 16, VmemSizeMb: 122 * 1024},
	"db.r4.8xlarge":  {VcpuCount: 32, VmemSizeMb: 244 * 1024},
	"db.r4.16xlarge": {VcpuCount: 64, VmemSizeMb: 488 * 1024},

	"db.x2g.medium":   {VcpuCount: 1, VmemSizeMb: 16 * 1024},
	"db.x2g.large":    {VcpuCount: 2, VmemSizeMb: 32 * 1024},
	"db.x2g.xlarge":   {VcpuCount: 4, VmemSizeMb: 64 * 1024},
	"db.x2g.2xlarge":  {VcpuCount: 8, VmemSizeMb: 128 * 1024},
	"db.x2g.4xlarge":  {VcpuCount: 16, VmemSizeMb: 256 * 1024},
	"db.x2g.8xlarge":  {VcpuCount: 32, VmemSizeMb: 512 * 1024},
	"db.x2g.12xlarge": {VcpuCount: 48, VmemSizeMb: 768 * 1024},
	"db.x2g.16xlarge": {VcpuCount: 64, VmemSizeMb: 1024 * 1024},
}
