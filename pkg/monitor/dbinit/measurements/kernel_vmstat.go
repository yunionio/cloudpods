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

package measurements

import "yunion.io/x/onecloud/pkg/apis/monitor"

var kernelVmstat = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "kernel_vmstat",
			DisplayName:  "kernel vmstat",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "nr_free_pages",
			DisplayName: "nr_free_pages",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_inactive_anon",
			DisplayName: "(integer, nr_inactive_anon)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_active_anon",
			DisplayName: "(integer, nr_active_anon)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_inactive_file",
			DisplayName: "(integer, nr_inactive_file)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_active_file",
			DisplayName: "(integer, nr_active_file)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_unevictable",
			DisplayName: "(integer, nr_unevictable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_mlock",
			DisplayName: "(integer, nr_mlock)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_anon_pages",
			DisplayName: "(integer, nr_anon_pages)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_mapped",
			DisplayName: "(integer, nr_mapped)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_file_pages",
			DisplayName: "(integer, nr_file_pages)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_dirty",
			DisplayName: "(integer, nr_dirty)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_writeback",
			DisplayName: "(integer, nr_writeback)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_slab_reclaimable",
			DisplayName: "(integer, nr_slab_reclaimable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_slab_unreclaimable",
			DisplayName: "(integer, nr_slab_unreclaimable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_page_table_pages",
			DisplayName: "(integer, nr_page_table_pages)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_kernel_stack",
			DisplayName: "(integer, nr_kernel_stack)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_unstable",
			DisplayName: "(integer, nr_unstable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_bounce",
			DisplayName: "(integer, nr_bounce)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_vmscan_write",
			DisplayName: "(integer, nr_vmscan_write)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_writeback_temp",
			DisplayName: "(integer, nr_writeback_temp)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_isolated_anon",
			DisplayName: "(integer, nr_isolated_anon)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_isolated_file",
			DisplayName: "(integer, nr_isolated_file)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_shmem",
			DisplayName: "(integer, nr_shmem)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "numa_hit",
			DisplayName: "(integer, numa_hit)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "numa_miss",
			DisplayName: "(integer, numa_miss)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "numa_foreign",
			DisplayName: "(integer, numa_foreign)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "numa_interleave",
			DisplayName: "(integer, numa_interleave)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "numa_local",
			DisplayName: "(integer, numa_local)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "numa_other",
			DisplayName: "(integer, numa_other)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "nr_anon_transparent_hugepages",
			DisplayName: "(integer, nr_anon_transparent_hugepages)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgpgin",
			DisplayName: "(integer, pgpgin)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgpgout",
			DisplayName: "(integer, pgpgout)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pswpin",
			DisplayName: "(integer, pswpin)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pswpout",
			DisplayName: "(integer, pswpout)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgalloc_dma",
			DisplayName: "(integer, pgalloc_dma)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgalloc_dma32",
			DisplayName: "(integer, pgalloc_dma32)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgalloc_normal",
			DisplayName: "(integer, pgalloc_normal)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgalloc_movable",
			DisplayName: "(integer, pgalloc_movable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgfree",
			DisplayName: "(integer, pgfree)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgactivate",
			DisplayName: "(integer, pgactivate)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgdeactivate",
			DisplayName: "(integer, pgdeactivate)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgfault",
			DisplayName: "(integer, pgfault)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgmajfault",
			DisplayName: "(integer, pgmajfault)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgrefill_dma",
			DisplayName: "(integer, pgrefill_dma)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgrefill_dma32",
			DisplayName: "(integer, pgrefill_dma32)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgrefill_normal",
			DisplayName: "(integer, pgrefill_normal)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgrefill_movable",
			DisplayName: "(integer, pgrefill_movable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgsteal_dma",
			DisplayName: "(integer, pgsteal_dma)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgsteal_dma32",
			DisplayName: "(integer, pgsteal_dma32)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgsteal_normal",
			DisplayName: "(integer, pgsteal_normal)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgsteal_movable",
			DisplayName: "(integer, pgsteal_movable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgscan_kswapd_dma",
			DisplayName: "(integer, pgscan_kswapd_dma)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgscan_kswapd_dma32",
			DisplayName: "(integer, pgscan_kswapd_dma32)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgscan_kswapd_normal",
			DisplayName: "(integer, pgscan_kswapd_normal)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgscan_kswapd_movable",
			DisplayName: "(integer, pgscan_kswapd_movable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgscan_direct_dma",
			DisplayName: "(integer, pgscan_direct_dma)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgscan_direct_dma32",
			DisplayName: "(integer, pgscan_direct_dma32)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgscan_direct_normal",
			DisplayName: "(integer, pgscan_direct_normal)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgscan_direct_movable",
			DisplayName: "(integer, pgscan_direct_movable)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "zone_reclaim_failed",
			DisplayName: "(integer, zone_reclaim_failed)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pginodesteal",
			DisplayName: "(integer, pginodesteal)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "slabs_scanned",
			DisplayName: "(integer, slabs_scanned)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "kswapd_steal",
			DisplayName: "(integer, kswapd_steal)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "kswapd_inodesteal",
			DisplayName: "(integer, kswapd_inodesteal)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "kswapd_low_wmark_hit_quickly",
			DisplayName: "(integer, kswapd_low_wmark_hit_quickly)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "kswapd_high_wmark_hit_quickly",
			DisplayName: "(integer, kswapd_high_wmark_hit_quickly)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "kswapd_skip_congestion_wait",
			DisplayName: "(integer, kswapd_skip_congestion_wait)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pageoutrun",
			DisplayName: "(integer, pageoutrun)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "allocstall",
			DisplayName: "(integer, allocstall)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "pgrotated",
			DisplayName: "(integer, pgrotated)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "compact_blocks_moved",
			DisplayName: "(integer, compact_blocks_moved)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "compact_pages_moved",
			DisplayName: "(integer, compact_pages_moved)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "compact_pagemigrate_failed",
			DisplayName: "(integer, compact_pagemigrate_failed)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "compact_stall",
			DisplayName: "(integer, compact_stall)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "compact_fail",
			DisplayName: "(integer, compact_fail)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "compact_success",
			DisplayName: "(integer, compact_success)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "htlb_buddy_alloc_success",
			DisplayName: "(integer, htlb_buddy_alloc_success)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "htlb_buddy_alloc_fail",
			DisplayName: "(integer, htlb_buddy_alloc_fail)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "unevictable_pgs_culled",
			DisplayName: "(integer, unevictable_pgs_culled)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "unevictable_pgs_scanned",
			DisplayName: "(integer, unevictable_pgs_scanned)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "unevictable_pgs_rescued",
			DisplayName: "(integer, unevictable_pgs_rescued)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "unevictable_pgs_mlocked",
			DisplayName: "(integer, unevictable_pgs_mlocked)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "unevictable_pgs_munlocked",
			DisplayName: "(integer, unevictable_pgs_munlocked)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "unevictable_pgs_cleared",
			DisplayName: "(integer, unevictable_pgs_cleared)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "unevictable_pgs_stranded",
			DisplayName: "(integer, unevictable_pgs_stranded)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "unevictable_pgs_mlockfreed",
			DisplayName: "(integer, unevictable_pgs_mlockfreed)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "thp_fault_alloc",
			DisplayName: "(integer, thp_fault_alloc)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "thp_fault_fallback",
			DisplayName: "(integer, thp_fault_fallback)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "thp_collapse_alloc",
			DisplayName: "(integer, thp_collapse_alloc)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "thp_collapse_alloc_failed",
			DisplayName: "(integer, thp_collapse_alloc_failed)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
		{
			Name:        "thp_split",
			DisplayName: "(integer, thp_split)",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}
