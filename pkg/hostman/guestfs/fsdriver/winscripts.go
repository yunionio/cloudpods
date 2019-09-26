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

package fsdriver

const WinScriptChangePassword = `

$username = $args[0]
$passwd = $args[1]
$loghash = $args[2]
$logpath = $args[3]
Function ChangePassword($u, $p) {
    $admin = [adsi]("WinNT://./$($u), user")
    $succ = 0
    $tried = 0
    $max_tries = 10
    while (($succ -eq 0) -and ($tried -lt $max_tries)) {
        Try {
            $admin.psbase.invoke("SetPassword", $p)
            $admin.psbase.CommitChanges()
            $succ = 1
        } Catch {
            Start-Sleep -s 1
        }
        $tried = $tried + 1
    }
}
if ($username -and $passwd) {
    if ($logpath) {
        "starting $loghash" | Out-File $logpath -Append -Encoding Default
        ChangePassword $username $passwd 2>&1 | Out-File $logpath -Append -Encoding Default
    } else {
        ChangePassword $username $passwd 2>&1 | Out-Null
    }
}

`

const WinScriptMountDisk = `

var MTW_GLOBAL_FSO = new ActiveXObject('Scripting.FileSystemObject');
var MTW_SCRIPT_PATH = mtw_gen_script_path();
var MTW_DEBUG_STREAM = null;

function mtw_create_shell() {
    return new ActiveXObject('WScript.Shell');
}

function mtw_gen_script_path() {
    var fso = MTW_GLOBAL_FSO;
    var folder = fso.GetSpecialFolder(2); // TemporaryFolder
    var folder_path = folder + '';
    var script_name = [];
    if (!/[\\\/]$/.test(folder_path)) {
        folder_path += '\\';
    }
    for (var i = 0; i < 5; i++) {
        script_name.push(mtw_gen_random_str(6));
    }
    return folder_path + script_name.join('-');
}

function mtw_gen_random_str(length) {
    var offset, result = [];
    var charcode_base = 'A'.charCodeAt(0);
    for (var i = 0; i < length; i++) {
        offset = Math.floor(Math.random() * 26);
        result.push(String.fromCharCode(charcode_base + offset));
    }
    return result.join('');
}

function mtw_prepare_debug(file_path) {
    var fso = new ActiveXObject('Scripting.FileSystemObject');
    var stream = fso.OpenTextFile(file_path, 8, true); // 8: ForAppending

    stream.WriteLine('');
    stream.WriteLine('================ ' + new Date() + ' ================');
    stream.WriteLine('');

    MTW_DEBUG_STREAM = stream;
}

function mtw_append_debug(cmd_lines, result_lines) {
    var i, len, stream = MTW_DEBUG_STREAM;

    if (!stream) return;

    stream.WriteLine('');
    stream.WriteLine('---------- script:');
    for (i = 0, len = cmd_lines.length; i < len; i++) {
        stream.WriteLine(cmd_lines[i]);
    }

    stream.WriteLine('---------- result:');
    for (i = 0, len = result_lines.length; i < len; i++) {
        stream.WriteLine(result_lines[i]);
    }
}

function mtw_execute_diskpart(cmd_lines) {
    var result_lines = [];
    var fso = MTW_GLOBAL_FSO;
    var stream, shell, exec_cmd;

    cmd_lines.push('exit');
    stream = fso.CreateTextFile(MTW_SCRIPT_PATH, true);
    for (var i = 0, len = cmd_lines.length; i < len; i++) {
        stream.WriteLine(cmd_lines[i]);
    }
    stream.close();

    shell = mtw_create_shell();
    exec_cmd = shell.Exec('diskpart /s ' + MTW_SCRIPT_PATH);
    while (exec_cmd.Status == 0) {
        WScript.Sleep(100);
    }
    stream = exec_cmd.StdOut;
    while (!stream.AtEndOfStream) {
        result_lines.push(stream.ReadLine());
    }
    fso.DeleteFile(MTW_SCRIPT_PATH);

    mtw_append_debug(cmd_lines, result_lines);

    return result_lines;
}

function mtw_get_disk_list() {
    var result_lines, line, match, disk_no;
    var disk_list = [];

    result_lines = mtw_execute_diskpart(['list disk']);
    for (var i = 0, len = result_lines.length; i < len; i++) {
        line = result_lines[i];
        /*
          Disk ###  Status         Size     Free     Dyn  Gpt
          --------  -------------  -------  -------  ---  ---
          Disk 0    Online           20 GB      0 B
          Disk 1    Offline          10 GB      0 B
        */
        match = line.match(/\s([1-9])\s\D+\s[1-9]\d*\s+[GMK]B\s+\d+\s+[GMK]?B/);
        if (match) {
            disk_no = match[1];
            disk_list.push({
                'disk_no': disk_no,
                'partition_list': mtw_get_partition_list(disk_no)
            });
        }
    }

    return disk_list;
}

function mtw_get_partition_list(disk_no) {
    var result_lines, line, match, partition_no;
    var partition_list = [];

    result_lines = mtw_execute_diskpart([
        'select disk=' + disk_no,
        'list partition'
    ]);
    for (var i = 0, len = result_lines.length; i < len; i++) {
        line = result_lines[i];
        /*
          Partition ###  Type              Size     Offset
          -------------  ----------------  -------  -------
          Partition 1    Primary              9 GB  1024 KB
        */
        match = line.match(/\s(\d)\s\D+\s[1-9]\d*\s+[GMK]B\s+\d+\s+[GMK]?B/);
        if (match) {
            partition_no = match[1];
            partition_list.push({
                'partition_no': partition_no,
                'partition_type': mtw_get_partition_type(disk_no, partition_no)
            });
        }
    }

    return partition_list;
}

var MTW_PARTITION_TYPE_NODATA = 'nodata';
var MTW_PARTITION_TYPE_INVALID = 'invalid';
function mtw_get_partition_type(disk_no, partition_no) {
    var result_lines, line, match;
    var possible_type_list = [];

    result_lines = mtw_execute_diskpart([
        'select disk=' + disk_no,
        'select partition=' + partition_no,
        'detail partition'
    ]);
    for (var i = 0, len = result_lines.length; i < len; i++) {
        line = result_lines[i];
        /*
        Partition 1
        Type  : 06
        Hidden: No
        Active: No
        Offset in Bytes: 1048576

          Volume ###  Ltr  Label        Fs     Type        Size     Status     Info
          ----------  ---  -----------  -----  ----------  -------  ---------  --------
        * Volume 3                      RAW    Partition      9 GB  Healthy
        */
        match = line.match(/:\s*([0-9a-f]{2})\b/i);
        if (match) {
            possible_type_list.push(match[1]);
        }
    }

    switch (possible_type_list.length) {
    case 0:
        return MTW_PARTITION_TYPE_NODATA;
    case 1:
        return possible_type_list[0];
    default:
        break;
    }
    return MTW_PARTITION_TYPE_INVALID;
}

function mtw_get_volume_list(disk_no) {
    var result_lines, line, match;
    var sep_line_exist = false;
    var volume_list = [];

    result_lines = mtw_execute_diskpart([
        'select disk=' + disk_no,
        'detail disk'
    ]);
    for (var i = 0, len = result_lines.length; i < len; i++) {
        line = result_lines[i];
        /*
        Red Hat VirtIO SCSI Disk Device
        Disk ID: 0004B605
        Type   : SCSI
        Status : Online
        Path   : 0
        Target : 0
        LUN ID : 0
        Location Path : PCIROOT(0)#PCI(0600)#SCSI(P00T00L00)
        Current Read-only State : No
        Read-only  : No
        Boot Disk  : No
        Pagefile Disk  : No
        Hibernation File Disk  : No
        Crashdump Disk  : No
        Clustered Disk  : No

          Volume ###  Ltr  Label        Fs     Type        Size     Status     Info
          ----------  ---  -----------  -----  ----------  -------  ---------  --------
          Volume 3                      RAW    Partition      9 GB  Healthy
        */
        if (!sep_line_exist) {
            match = line.match(/-+\s+-+\s+-+\s+-+/);
            if (match) {
                sep_line_exist = true;
            }
        } else {
            match = line.match(/\s(\d)\s.+\s\d+\s+[GMK]?B/);
            if (match) {
                volume_list.push({'volume_no': match[1]});
            }
        }
    }

    return volume_list;
}

function mtw_assign_volume_letter(volume_no_set, letter_offset) {
    var result_lines, line, match, i, len;
    var volume_no, letter, charcode, charcode_max, letter_set;
    var volume_map = {}, volume_shift_list = [], cmd_lines = [];

    result_lines = mtw_execute_diskpart(['list volume']);
    for (i = 0, len = result_lines.length; i < len; i++) {
        line = result_lines[i];
        /*
          Volume ###  Ltr  Label        Fs     Type        Size     Status     Info
          ----------  ---  -----------  -----  ----------  -------  ---------  --------
          Volume 0     D                       CD-ROM          0 B  No Media
          Volume 1         ????         NTFS   Partition    100 MB  Healthy    System
          Volume 2     C                NTFS   Partition     19 GB  Healthy    Boot
          Volume 3                      RAW    Partition      9 GB  Healthy
        */
        match = line.match(/\s(\d)\s+([D-Z])\s.+\d+\s+[GMK]?B/i);
        if (match) {
            volume_no = match[1];
            letter = match[2].toUpperCase();
            if (volume_map.hasOwnProperty(letter)) {
                return false;
            }
            volume_map[letter] = volume_no;
        }
    }

    charcode = 'D'.charCodeAt(0);
    charcode += letter_offset;
    letter_set = String.fromCharCode(charcode);
    if (volume_map.hasOwnProperty(letter_set) && volume_map[letter_set] == volume_no_set) {
        return true;
    }
    charcode_max = 'Z'.charCodeAt(0);
    while (true) {
        letter = String.fromCharCode(charcode);
        if (!volume_map.hasOwnProperty(letter)) {
            break;
        }
        if (charcode >= charcode_max) {
            return false;
        }
        volume_shift_list.push({
            'volume_no': volume_map[letter],
            'charcode_next': charcode + 1
        });
        charcode++;
    }
    if (volume_shift_list.length > 0) {
        volume_shift_list.sort(function(a, b) {
            return b.charcode_next - a.charcode_next;
        });
        for (i = 0, len = volume_shift_list.length; i < len; i++) {
            volume_no = volume_shift_list[i].volume_no;
            charcode = volume_shift_list[i].charcode_next;
            cmd_lines.push(
                'select volume=' + volume_no,
                'assign letter=' + String.fromCharCode(charcode),
                'select partition 1',
                'format fs ntfs quick'
            );
        }
    }
    cmd_lines.push(
        'select volume=' + volume_no_set,
        'assign letter=' + letter_set,
        'select partition 1',
        'format fs ntfs quick'
    );
    mtw_execute_diskpart(cmd_lines);

    return true;
}

function mtw_wait_loop(total_ms, step_ms, callback) {
    while (true) {
        if (callback()) {
            break;
        }
        if (total_ms < step_ms) {
            break;
        }
        total_ms -= step_ms;
        WScript.Sleep(step_ms);
    }
}

function mtw_get_disk_list_wait() {
    var disk_list_ret = [];

    /* http://support.microsoft.com/kb/870912 */
    mtw_wait_loop(5000, 500, function() {
        var i, j, disk_list, disk, partition;
        disk_list = mtw_get_disk_list();
        for (i = 0; i < disk_list.length; i++) {
            disk = disk_list[i];
            for (j = 0; j < disk.partition_list.length; j++) {
                partition = disk.partition_list[j];
                if (partition.partition_type == MTW_PARTITION_TYPE_NODATA) {
                    return false;
                }
            }
        }
        disk_list_ret = disk_list;
        return true;
    });

    return disk_list_ret;
}

function mtw_get_volume_list_wait(disk_no) {
    var volume_list_ret = [];

    mtw_wait_loop(5000, 500, function() {
        var volume_list = mtw_get_volume_list(disk_no);
        if (volume_list.length > 0) {
            volume_list_ret = volume_list;
            return true;
        }
        return false;
    });

    return volume_list_ret;
}

function mtw_mount_disk() {
    var disk_list, disk_list_mounted, disk, partition, volume_list;
    var do_mount, do_create, do_delete;
    var cmd_lines, letter_offset = 0;

    disk_list = mtw_get_disk_list_wait();
    disk_list_mounted = [];
    for (var i = 0, len = disk_list.length; i < len; i++) {
        disk = disk_list[i];
        partition = null;
        do_mount = do_create = do_delete = false;
        if (disk.partition_list.length == 0) {
            do_mount = true;
            do_create = true;
        } else if (disk.partition_list.length == 1) {
            partition = disk.partition_list[0];
            switch (partition.partition_type) {
            case '06': // DOS 3.31+ 16-bit FAT (over 32M)
            case '07': // Windows NT NTFS
                do_mount = true;
                break;
            case '83': // Linux native partition
                do_mount = true;
                do_delete = true;
                do_create = true;
                break;
            default:
                break;
            }
        }
        if (!do_mount) {
            continue;
        }
        cmd_lines = [
            'select disk=' + disk.disk_no,
            'online disk',
            'attributes disk clear readonly'
        ];
        mtw_execute_diskpart(cmd_lines);
        if (do_create) {
            cmd_lines = ['select disk=' + disk.disk_no];
            if (partition && do_delete) {
                cmd_lines.push(
                    'select partition=' + partition.partition_no,
                    'delete partition'
                );
            }
            cmd_lines.push('create partition primary');
            mtw_execute_diskpart(cmd_lines);
        }
        disk_list_mounted.push(disk);
    }

    for (i = 0, len = disk_list_mounted.length; i < len; i++) {
        disk = disk_list_mounted[i];
        if (i == 0) {
            volume_list = mtw_get_volume_list_wait(disk.disk_no);
        } else {
            volume_list = mtw_get_volume_list(disk.disk_no);
        }
        if (volume_list.length == 1) {
            mtw_assign_volume_letter(volume_list[0].volume_no, letter_offset);
            letter_offset += 1;
        }
    }
}

function mtw_extend_c() {
    cmd_lines = [
        'select volume c',
        'extend',
    ];
    mtw_execute_diskpart(cmd_lines);
    mtw_append_debug(["extend c"], ["success"]);
}

function mtw_main() {
    var exec_helper, args = WScript.Arguments, debug_path = '';

    for (var i = 0, len = args.length; i < len; i++) {
        if (args(i) == '--debug') {
            if (i < len) {
                i += 1;
                debug_path = args(i);
            }
        }
    }

    if (debug_path) {
        mtw_prepare_debug(debug_path);
    }

    /* http://support.microsoft.com/kb/937252 */
    exec_helper = mtw_create_shell().Exec('diskpart');
    try {
        mtw_extend_c();
    } catch (e) {
        mtw_append_debug(["extend c"], ["failed"]);
    }
    try {
        mtw_mount_disk();
    } catch (e) {
        // nothing
    }
    exec_helper.StdIn.WriteLine('exit');
    while (exec_helper.Status == 0) {
        WScript.Sleep(100);
    }
}

mtw_main();

`
