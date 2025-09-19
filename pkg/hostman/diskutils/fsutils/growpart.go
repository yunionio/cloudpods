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

package fsutils

var GrowPartScript = `
#!/bin/sh
#    Copyright (C) 2011 Canonical Ltd.
#    Copyright (C) 2013 Hewlett-Packard Development Company, L.P.
#
#    Authors: Scott Moser <smoser@canonical.com>
#             Juerg Haefliger <juerg.haefliger@hp.com>
#
#    This program is free software: you can redistribute it and/or modify
#    it under the terms of the GNU General Public License as published by
#    the Free Software Foundation, version 3 of the License.
#
#    This program is distributed in the hope that it will be useful,
#    but WITHOUT ANY WARRANTY; without even the implied warranty of
#    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#    GNU General Public License for more details.
#
#    You should have received a copy of the GNU General Public License
#    along with this program.  If not, see <http://www.gnu.org/licenses/>.

# the fudge factor. if within this many bytes dont bother
FUDGE=${GROWPART_FUDGE:-$((1024*1024))}
TEMP_D=""
RESTORE_FUNC=""
RESTORE_HUMAN=""
VERBOSITY=0
DISK=""
PART=""
PT_UPDATE=false
DRY_RUN=0

SFDISK_VERSION=""
SFDISK_2_26="22600"
SFDISK_V_WORKING_GPT="22603"
MBR_BACKUP=""
GPT_BACKUP=""
_capture=""

error() {
        echo "$@" 1>&2
}

fail() {
        [ $# -eq 0 ] || echo "FAILED:" "$@"
        exit 2
}

nochange() {
        echo "NOCHANGE:" "$@"
        exit 1
}

changed() {
        echo "CHANGED:" "$@"
        exit 0
}

change() {
        echo "CHANGE:" "$@"
        exit 0
}

cleanup() {
        if [ -n "${RESTORE_FUNC}" ]; then
                error "***** WARNING: Resize failed, attempting to revert ******"
                if ${RESTORE_FUNC} ; then
                        error "***** Restore appears to have gone OK ****"
                else
                        error "***** Restore FAILED! ******"
                        if [ -n "${RESTORE_HUMAN}" -a -f "${RESTORE_HUMAN}" ]; then
                                error "**** original table looked like: ****"
                                cat "${RESTORE_HUMAN}" 1>&2
                        else
                                error "We seem to have not saved the partition table!"
                        fi
                fi
        fi
        [ -z "${TEMP_D}" -o ! -d "${TEMP_D}" ] || rm -Rf "${TEMP_D}"
}

debug() {
        local level=${1}
        shift
        [ "${level}" -gt "${VERBOSITY}" ] && return
        if [ "${DEBUG_LOG}" ]; then
                echo "$@" >>"${DEBUG_LOG}"
        else
                error "$@"
        fi
}

debugcat() {
        local level="$1"
        shift;
        [ "${level}" -gt "$VERBOSITY" ] && return
        if [ "${DEBUG_LOG}" ]; then
                cat "$@" >>"${DEBUG_LOG}"
        else
                cat "$@" 1>&2
        fi
}

mktemp_d() {
        # just a mktemp -d that doens't need mktemp if its not there.
        _RET=$(mktemp -d "${TMPDIR:-/tmp}/${0##*/}.XXXXXX" 2>/dev/null) &&
                return
        _RET=$(umask 077 && t="${TMPDIR:-/tmp}/${0##*/}.$$" &&
                mkdir "${t}" && echo "${t}")
        return
}

Usage() {
        cat <<EOF
${0##*/} disk partition
   rewrite partition table so that partition takes up all the space it can
   options:
    -h | --help       print Usage and exit
         --fudge F    if part could be resized, but change would be
                      less than 'F' bytes, do not resize (default: ${FUDGE})
    -N | --dry-run    only report what would be done, show new 'sfdisk -d'
    -v | --verbose    increase verbosity / debug
    -u | --update  R  update the the kernel partition table info after growing
                      this requires kernel support and 'partx --update'
                      R is one of:
                       - 'auto'  : [default] update partition if possible
                       - 'force' : try despite sanity checks (fail on failure)
                       - 'off'   : do not attempt
                       - 'on'    : fail if sanity checks indicate no support

   Example:
    - ${0##*/} /dev/sda 1
      Resize partition 1 on /dev/sda
EOF
}

bad_Usage() {
        Usage 1>&2
        error "$@"
        exit 2
}

sfdisk_restore_legacy() {
        sfdisk --no-reread "${DISK}" -I "${MBR_BACKUP}"
}

sfdisk_restore() {
        # files are named: sfdisk-<device>-<offset>.bak
        local f="" offset="" fails=0
        for f in "${MBR_BACKUP}"*.bak; do
                [ -f "$f" ] || continue
                offset=${f##*-}
                offset=${offset%.bak}
                [ "$offset" = "$f" ] && {
                        error "WARN: confused by file $f";
                        continue;
                }
                dd "if=$f" "of=${DISK}" seek=$(($offset)) bs=1 conv=notrunc ||
                        { error "WARN: failed restore from $f"; fails=$(($fails+1)); }
        done
        return $fails
}

sfdisk_worked_but_blkrrpart_failed() {
        local ret="$1" output="$2"
        # exit code found was just 1, but dont insist on that
        #[ $ret -eq 1 ] || return 1
        # Successfully wrote the new partition table
        if grep -qi "Success.* wrote.* new.* partition" "$output"; then
                grep -qi "BLKRRPART: Device or resource busy" "$output"
                return
        # The partition table has been altered.
        elif grep -qi "The.* part.* table.* has.* been.* altered" "$output"; then
                # Re-reading the partition table failed
                grep -qi "Re-reading.* partition.* table.* failed" "$output"
                return
        fi
        return $ret
}

get_sfdisk_version() {
        # set SFDISK_VERSION to MAJOR*10000+MINOR*100+MICRO
        local out oifs="$IFS" ver=""
        [ -n "$SFDISK_VERSION" ] && return 0
        # expected output: sfdisk from util-linux 2.25.2
        out=$(LANG=C sfdisk --version) ||
                { error "failed to get sfdisk version"; return 1; }
        set -- $out
        ver=$4
        case "$ver" in
                [0-9]*.[0-9]*.[0-9]|[0-9].[0-9]*)
                        IFS="."; set -- $ver; IFS="$oifs"
                        SFDISK_VERSION=$(($1*10000+$2*100+${3:-0}))
                        return 0;;
                *) error "unexpected output in sfdisk --version [$out]"
                        return 1;;
        esac
}

resize_sfdisk() {
        local humanpt="${TEMP_D}/recovery"
        local mbr_backup="${TEMP_D}/orig.save"
        local restore_func=""
        local format="$1"

        local change_out=${TEMP_D}/change.out
        local dump_out=${TEMP_D}/dump.out
        local new_out=${TEMP_D}/new.out
        local dump_mod=${TEMP_D}/dump.mod
        local tmp="${TEMP_D}/tmp.out"
        local err="${TEMP_D}/err.out"
        local mbr_max_512="4294967296"

        local pt_start pt_size pt_end max_end new_size change_info dpart
        local sector_num sector_size disk_size tot out

        LANG=C rqe sfd_list sfdisk --list --unit=S "$DISK" >"$tmp" ||
                fail "failed: sfdisk --list $DISK"
        if [ "${SFDISK_VERSION}" -lt ${SFDISK_2_26} ]; then
                # exected output contains: Units: sectors of 512 bytes, ...
                out=$(awk '$1 == "Units:" && $5 ~ /bytes/ { print $4 }' "$tmp") ||
                        fail "failed to read sfdisk output"
                if [ -z "$out" ]; then
                        error "WARN: sector size not found in sfdisk output, assuming 512"
                        sector_size=512
                else
                        sector_size="$out"
                fi
                local _w _cyl _w1 _heads _w2 sectors _w3 t s
                # show-size is in units of 1024 bytes (same as /proc/partitions)
                t=$(sfdisk --show-size "${DISK}") ||
                        fail "failed: sfdisk --show-size $DISK"
                disk_size=$((t*1024))
                sector_num=$(($disk_size/$sector_size))
                msg="disk size '$disk_size' not evenly div by sector size '$sector_size'"
                [ "$((${disk_size}%${sector_size}))" -eq 0 ] ||
                        error "WARN: $msg"
                restore_func=sfdisk_restore_legacy
        else
                # --list first line output:
                # Disk /dev/vda: 20 GiB, 21474836480 bytes, 41943040 sectors
                local _x
                read _x _x _x _x disk_size _x sector_num _x  < "$tmp"
                sector_size=$((disk_size/$sector_num))
                restore_func=sfdisk_restore
        fi

        debug 1 "$sector_num sectors of $sector_size. total size=${disk_size} bytes"

        rqe sfd_dump sfdisk --unit=S --dump "${DISK}" >"${dump_out}" ||
                fail "failed to dump sfdisk info for ${DISK}"
        RESTORE_HUMAN="$dump_out"

        {
                echo "## sfdisk --unit=S --dump ${DISK}"
                cat "${dump_out}"
        }  >"$humanpt"

        [ $? -eq 0 ] || fail "failed to save sfdisk -d output"
        RESTORE_HUMAN="$humanpt"

        debugcat 1 "$humanpt"

        sed -e 's/,//g; s/start=/start /; s/size=/size /' "${dump_out}" \
                >"${dump_mod}" ||
                fail "sed failed on dump output"

        dpart="${DISK}${PART}" # disk and partition number
        if [ -b "${DISK}p${PART}" -a "${DISK%[0-9]}" != "${DISK}" ]; then
                # for block devices that end in a number (/dev/nbd0)
                # the partition is "<name>p<partition_number>" (/dev/nbd0p1)
                dpart="${DISK}p${PART}"
        elif [ "${DISK#/dev/loop[0-9]}" != "${DISK}" ]; then
                # for /dev/loop devices, sfdisk output will be <name>p<number>
                # format also, even though there is not a device there.
                dpart="${DISK}p${PART}"
        fi

        pt_start=$(awk '$1 == pt { print $4 }' "pt=${dpart}" <"${dump_mod}") &&
                pt_size=$(awk '$1 == pt { print $6 }' "pt=${dpart}" <"${dump_mod}") &&
                [ -n "${pt_start}" -a -n "${pt_size}" ] &&
                pt_end=$((${pt_size}+${pt_start})) ||
                fail "failed to get start and end for ${dpart} in ${DISK}"

        # find the minimal starting location that is >= pt_end
        max_end=$(awk '$3 == "start" { if($4 >= pt_end && $4 < min)
                { min = $4 } } END { printf("%s\n",min); }' \
                min=${sector_num} pt_end=${pt_end} "${dump_mod}") &&
                [ -n "${max_end}" ] ||
                fail "failed to get max_end for partition ${PART}"

        if [ "$format" = "gpt" ]; then
                # sfdisk respects 'last-lba' in input, and complains about
                # partitions that go past that.  without it, it does the right thing.
                sed -i '/^last-lba:/d' "$dump_out" ||
                        fail "failed to remove last-lba from output"
        fi
        if [ "$format" = "dos" ]; then
                mbr_max_sectors=$((mbr_max_512*$((sector_size/512))))
                if [ "$max_end" -gt "$mbr_max_sectors" ]; then
                        max_end=$mbr_max_sectors
                fi
                [ $(($disk_size/512)) -gt $mbr_max_512 ] &&
                        debug 0 "WARNING: MBR/dos partitioned disk is larger than 2TB." \
                                "Additional space will go unused."
        fi

        local gpt_second_size="33"
        if [ "${max_end}" -gt "$((${sector_num}-${gpt_second_size}))" ]; then
                # if mbr allow subsequent conversion to gpt without shrinking the
                # partition.  safety net at cost of 33 sectors, seems reasonable.
                # if gpt, we can't write there anyway.
                debug 1 "padding ${gpt_second_size} sectors for gpt secondary header"
                max_end=$((${sector_num}-${gpt_second_size}))
        fi

        debug 1 "max_end=${max_end} tot=${sector_num} pt_end=${pt_end}" \
                "pt_start=${pt_start} pt_size=${pt_size}"
        [ $((${pt_end})) -eq ${max_end} ] &&
                nochange "partition ${PART} is size ${pt_size}. it cannot be grown"
        [ $((${pt_end}+(${FUDGE}/$sector_size))) -gt ${max_end} ] &&
                nochange "partition ${PART} could only be grown by" \
                "$((${max_end}-${pt_end})) [fudge=$((${FUDGE}/$sector_size))]"

        # now, change the size for this partition in ${dump_out} to be the
        # new size
        new_size=$((${max_end}-${pt_start}))
        sed "\|^\s*${dpart} |s/\(.*\)${pt_size},/\1${new_size},/" "${dump_out}" \
                >"${new_out}" ||
                fail "failed to change size in output"

        change_info="partition=${PART} start=${pt_start}"
        change_info="${change_info} old: size=${pt_size} end=${pt_end}"
        change_info="${change_info} new: size=${new_size} end=${max_end}"
        if [ ${DRY_RUN} -ne 0 ]; then
                echo "CHANGE: ${change_info}"
                {
                        echo "# === old sfdisk -d ==="
                        cat "${dump_out}"
                        echo "# === new sfdisk -d ==="
                        cat "${new_out}"
                } 1>&2
                exit 0
        fi

        MBR_BACKUP="${mbr_backup}"
        LANG=C sfdisk --no-reread "${DISK}" --force \
                -O "${mbr_backup}" <"${new_out}" >"${change_out}" 2>&1
        ret=$?
        [ $ret -eq 0 ] || RESTORE_FUNC="${restore_func}"

        if [ $ret -eq 0 ]; then
                debug 1 "resize of ${DISK} returned 0."
                if [ $VERBOSITY -gt 2 ]; then
                        sed 's,^,| ,' "${change_out}" 1>&2
                fi
        elif $PT_UPDATE &&
                sfdisk_worked_but_blkrrpart_failed "$ret" "${change_out}"; then
                # if the command failed, but it looks like only because
                # the device was busy and we have pt_update, then go on
                debug 1 "sfdisk failed, but likely only because of blkrrpart"
        else
                error "attempt to resize ${DISK} failed. sfdisk output below:"
                sed 's,^,| ,' "${change_out}" 1>&2
                fail "failed to resize"
        fi

        rq pt_update pt_update "$DISK" "$PART" ||
                fail "pt_resize failed"

        RESTORE_FUNC=""

        changed "${change_info}"

        # dump_out looks something like:
        ## partition table of /tmp/out.img
        #unit: sectors
        #
        #/tmp/out.img1 : start=        1, size=    48194, Id=83
        #/tmp/out.img2 : start=    48195, size=   963900, Id=83
        #/tmp/out.img3 : start=  1012095, size=   305235, Id=82
        #/tmp/out.img4 : start=  1317330, size=   771120, Id= 5
        #/tmp/out.img5 : start=  1317331, size=   642599, Id=83
        #/tmp/out.img6 : start=  1959931, size=    48194, Id=83
        #/tmp/out.img7 : start=  2008126, size=    80324, Id=83
}

gpt_restore() {
        sgdisk -l "${GPT_BACKUP}" "${DISK}"
}

resize_sgdisk() {
        GPT_BACKUP="${TEMP_D}/pt.backup"

        local pt_info="${TEMP_D}/pt.info"
        local pt_pretend="${TEMP_D}/pt.pretend"
        local pt_data="${TEMP_D}/pt.data"
        local out="${TEMP_D}/out"

        local dev="disk=${DISK} partition=${PART}"

        local pt_start pt_end pt_size last pt_max code guid name new_size
        local old new change_info sector_size

        # Dump the original partition information and details to disk. This is
        # used in case something goes wrong and human interaction is required
        # to revert any changes.
        rqe sgd_info sgdisk "--info=${PART}" --print "${DISK}" >"${pt_info}" ||
                fail "${dev}: failed to dump original sgdisk info"
        RESTORE_HUMAN="${pt_info}"

        sector_size=$(awk '$0 ~ /^Logical sector size:.*bytes/ { print $4 }' \
                "$pt_info") && [ -n "$sector_size" ] || {
                sector_size=512
                error "WARN: did not find sector size, assuming 512"
        }

        debug 1 "$dev: original sgdisk info:"
        debugcat 1 "${pt_info}"

        # Pretend to move the backup GPT header to the end of the disk and dump
        # the resulting partition information. We use this info to determine if
        # we have to resize the partition.
        rqe sgd_pretend sgdisk --pretend --move-second-header \
                --print "${DISK}" >"${pt_pretend}" ||
                fail "${dev}: failed to dump pretend sgdisk info"

        debug 1 "$dev: pretend sgdisk info"
        debugcat 1 "${pt_pretend}"

        # Extract the partition data from the pretend dump
        awk 'found { print } ; $1 == "Number" { found = 1 }' \
                "${pt_pretend}" >"${pt_data}" ||
                fail "${dev}: failed to parse pretend sgdisk info"

        # Get the start and end sectors of the partition to be grown
        pt_start=$(awk '$1 == '"${PART}"' { print $2 }' "${pt_data}") &&
                [ -n "${pt_start}" ] ||
                fail "${dev}: failed to get start sector"
        pt_end=$(awk '$1 == '"${PART}"' { print $3 }' "${pt_data}") &&
                [ -n "${pt_end}" ] ||
                fail "${dev}: failed to get end sector"
        # sgdisk start and end are inclusive.  start 2048 length 10 ends at 2057.
        pt_end=$((pt_end+1))
        pt_size="$((${pt_end} - ${pt_start}))"

        # Get the last usable sector
        last=$(awk '/last usable sector is/ { print $NF }' \
                "${pt_pretend}") && [ -n "${last}" ] ||
                fail "${dev}: failed to get last usable sector"

        # Find the minimal start sector that is >= pt_end
        pt_max=$(awk '{ if ($2 >= pt_end && $2 < min) { min = $2 } } END \
                { print min }' min="${last}" pt_end="${pt_end}" \
                "${pt_data}") && [ -n "${pt_max}" ] ||
                fail "${dev}: failed to find max end sector"

        debug 1 "${dev}: pt_start=${pt_start} pt_end=${pt_end}" \
                "pt_size=${pt_size} pt_max=${pt_max} last=${last}"

        # Check if the partition can be grown
        [ "${pt_end}" -eq "${pt_max}" ] &&
                nochange "${dev}: size=${pt_size}, it cannot be grown"
        [ "$((${pt_end} + ${FUDGE}/${sector_size}))" -gt "${pt_max}" ] &&
                nochange "${dev}: could only be grown by" \
                "$((${pt_max} - ${pt_end})) [fudge=$((${FUDGE}/$sector_size))]"

        # The partition can be grown if we made it here. Get some more info
        # about it so we can do it properly.
        # FIXME: Do we care about the attribute flags?
        code=$(awk '/^Partition GUID code:/ { print $4 }' "${pt_info}")
        guid=$(awk '/^Partition unique GUID:/ { print $4 }' "${pt_info}")
        name=$(awk '/^Partition name:/ { gsub(/'"'"'/, "") ; \
                if (NF >= 3) print substr($0, index($0, $3)) }' "${pt_info}")
        [ -n "${code}" -a -n "${guid}" ] ||
                fail "${dev}: failed to parse sgdisk details"

        debug 1 "${dev}: code=${code} guid=${guid} name='${name}'"
        local wouldrun=""
        [ "$DRY_RUN" -ne 0 ] && wouldrun="would-run"

        # Calculate the new size of the partition
        new_size=$((${pt_max} - ${pt_start}))
        change_info="partition=${PART} start=${pt_start}"
        change_info="${change_info} old: size=${pt_size} end=${pt_end}"
        change_info="${change_info} new: size=${new_size} end=${pt_max}"

        # Backup the current partition table, we're about to modify it
        rq sgd_backup $wouldrun sgdisk "--backup=${GPT_BACKUP}" "${DISK}" ||
                fail "${dev}: failed to backup the partition table"

        # Modify the partition table. We do it all in one go (the order is
        # important!):
        #  - move the GPT backup header to the end of the disk
        #  - delete the partition
        #  - recreate the partition with the new size
        #  - set the partition code
        #  - set the partition GUID
        #  - set the partition name
        rq sgdisk_mod $wouldrun sgdisk --move-second-header "--delete=${PART}" \
                "--new=${PART}:${pt_start}:$((pt_max-1))" \
                "--typecode=${PART}:${code}" \
                "--partition-guid=${PART}:${guid}" \
                "--change-name=${PART}:${name}" "${DISK}" &&
                rq pt_update $wouldrun pt_update "$DISK" "$PART" || {
                RESTORE_FUNC=gpt_restore
                fail "${dev}: failed to repartition"
        }

        # Dry run
        [ "${DRY_RUN}" -ne 0 ] && change "${change_info}"

        changed "${change_info}"
}

kver_to_num() {
        local kver="$1" maj="" min="" mic="0"
        kver=${kver%%-*}
        maj=${kver%%.*}
        min=${kver#${maj}.}
        min=${min%%.*}
        mic=${kver#${maj}.${min}.}
        [ "$kver" = "$mic" ] && mic=0
        _RET=$(($maj*1000*1000+$min*1000+$mic))
}

kver_cmp() {
        local op="$2" n1="" n2=""
        kver_to_num "$1"
        n1="$_RET"
        kver_to_num "$3"
        n2="$_RET"
        [ $n1 $op $n2 ]
}

rq() {
        # runquieterror(label, command)
        # gobble stderr of a command unless it errors
        local label="$1" ret="" efile=""
        efile="$TEMP_D/$label.err"
        shift;

        local rlabel="running"
        [ "$1" = "would-run" ] && rlabel="would-run" && shift

        local cmd="" x=""
        for x in "$@"; do
                [ "${x#* }" != "$x" -o "${x#* \"}" != "$x" ] && x="'$x'"
                cmd="$cmd $x"
        done
        cmd=${cmd# }

        debug 2 "$rlabel[$label][$_capture]" "$cmd"
        [ "$rlabel" = "would-run" ] && return 0

        if [ "${_capture}" = "erronly" ]; then
                "$@" 2>"$TEMP_D/$label.err"
                ret=$?
        else
                "$@" >"$TEMP_D/$label.err" 2>&1
                ret=$?
        fi
        if [ $ret -ne 0 ]; then
                error "failed [$label:$ret]" "$@"
                cat "$efile" 1>&2
        fi
        return $ret
}

rqe() {
        local _capture="erronly"
        rq "$@"
}

verify_ptupdate() {
        local input="$1" found="" reason="" kver=""

        # we can always satisfy 'off'
        if [ "$input" = "off" ]; then
                _RET="false";
                return 0;
        fi

        if command -v partx >/dev/null 2>&1; then
                local out="" ret=0
                out=$(partx --help 2>&1)
                ret=$?
                if [ $ret -eq 0 ]; then
                        echo "$out" | grep -q -- --update || {
                                reason="partx has no '--update' flag in usage."
                                found="off"
                        }
                else
                        reason="'partx --help' returned $ret. assuming it is old."
                        found="off"
                fi
        else
                reason="no 'partx' command"
                found="off"
        fi

        if [ -z "$found" ]; then
                if [ "$(uname)" != "Linux" ]; then
                        reason="Kernel is not Linux per uname."
                        found="off"
                fi
        fi

        if [ -z "$found" ]; then
                kver=$(uname -r) || debug 1 "uname -r failed!"

                if ! kver_cmp "${kver-0.0.0}" -ge 3.8.0; then
                        reason="Kernel '$kver' < 3.8.0."
                        found="off"
                fi
        fi

        if [ -z "$found" ]; then
                _RET="true"
                return 0
        fi

        case "$input" in
                on) error "$reason"; return 1;;
                auto)
                        _RET="false";
                        debug 1 "partition update disabled: $reason"
                        return 0;;
                force)
                        _RET="true"
                        error "WARNING: ptupdate forced on even though: $reason"
                        return 0;;
        esac
        error "unknown input '$input'";
        return 1;
}

pt_update() {
        local dev="$1" part="$2" update="${3:-$PT_UPDATE}"
        if ! $update; then
                return 0
        fi
        # partx only works on block devices (do not run on file)
        [ -b "$dev" ] || return 0
        partx --update --nr "$part" "$dev"
}

has_cmd() {
        command -v "${1}" >/dev/null 2>&1
}

resize_sgdisk_gpt() {
        resize_sgdisk gpt
}

resize_sgdisk_dos() {
        fail "unable to resize dos label with sgdisk"
}

resize_sfdisk_gpt() {
        resize_sfdisk gpt
}

resize_sfdisk_dos() {
        resize_sfdisk dos
}

get_table_format() {
        local out="" disk="$1"
        if has_cmd blkid && out=$(blkid -o value -s PTTYPE "$disk") &&
                [ "$out" = "dos" -o "$out" = "gpt" ]; then
                _RET="$out"
                return
        fi
        _RET="dos"
        if [ ${SFDISK_VERSION} -lt ${SFDISK_2_26} ] &&
                out=$(sfdisk --id --force "$disk" 1 2>/dev/null); then
                if [ "$out" = "ee" ]; then
                        _RET="gpt"
                else
                        _RET="dos"
                fi
                return
        elif out=$(LANG=C sfdisk --list "$disk"); then
                out=$(echo "$out" | sed -e '/Disklabel type/!d' -e 's/.*: //')
                case "$out" in
                        gpt|dos) _RET="$out";;
                        *) error "WARN: unknown label $out";;
                esac
        fi
}

get_resizer() {
        local format="$1" user=${2:-"auto"}

        case "$user" in
                sgdisk) _RET="resize_sgdisk_$format"; return;;
                sfdisk) _RET="resize_sfdisk_$format"; return;;
                auto) :;;
                *) error "unexpected input: '$user'";;
        esac

        if [ "$format" = "dos" ]; then
                _RET="resize_sfdisk_dos"
                return 0
        fi

        if [ "${SFDISK_VERSION}" -ge ${SFDISK_V_WORKING_GPT} ]; then
                # sfdisk 2.26.2 works for resize but loses type (LP: #1474090)
                _RET="resize_sfdisk_gpt"
        elif has_cmd sgdisk; then
                _RET="resize_sgdisk_$format"
        else
                error "no tools available to resize disk with '$format'"
                return 1
        fi
        return 0
}

pt_update="auto"
resizer=${GROWPART_RESIZER:-"auto"}
while [ $# -ne 0 ]; do
        cur=${1}
        next=${2}
        case "$cur" in
                -h|--help)
                        Usage
                        exit 0
                        ;;
                --fudge)
                        FUDGE=${next}
                        shift
                        ;;
                -N|--dry-run)
                        DRY_RUN=1
                        ;;
                -u|--update|--update=*)
                        if [ "${cur#--update=}" != "$cur" ]; then
                                next="${cur#--update=}"
                        else
                                shift
                        fi
                        case "$next" in
                                off|auto|force|on) pt_update=$next;;
                                *) fail "unknown --update option: $next";;
                        esac
                        ;;
                -v|--verbose)
                        VERBOSITY=$(($VERBOSITY+1))
                        ;;
                --)
                        shift
                        break
                        ;;
                -*)
                        fail "unknown option ${cur}"
                        ;;
                *)
                        if [ -z "${DISK}" ]; then
                                DISK=${cur}
                        else
                                [ -z "${PART}" ] || fail "confused by arg ${cur}"
                                PART=${cur}
                        fi
                        ;;
        esac
        shift
done

[ -n "${DISK}" ] || bad_Usage "must supply disk and partition-number"
[ -n "${PART}" ] || bad_Usage "must supply partition-number"

has_cmd "sfdisk" || fail "sfdisk not found"
get_sfdisk_version || fail

[ -e "${DISK}" ] || fail "${DISK}: does not exist"

# If $DISK is a symlink, resolve it.
# This avoids problems due to varying partition device name formats
# (e.g. "1" for /dev/sda vs "-part1" for /dev/disk/by-id/name)
if [ -L "${DISK}" ]; then
        has_cmd readlink ||
                fail "${DISK} is a symlink, but 'readlink' command not available."
        real_disk=$(readlink -f "${DISK}") || fail "unable to resolve ${DISK}"
        debug 1 "${DISK} resolved to ${real_disk}"
        DISK=${real_disk}
fi

[ "${PART#*[!0-9]}" = "${PART}" ] || fail "partition-number must be a number"

verify_ptupdate "$pt_update" || fail
PT_UPDATE=$_RET

debug 1 "update-partition set to $PT_UPDATE"

mktemp_d && TEMP_D="${_RET}" || fail "failed to make temp dir"
trap cleanup 0 # EXIT - some shells may not like 'EXIT' but are ok with 0

# get the ID of the first partition to determine if it's MBR or GPT
get_table_format "$DISK" || fail
format=$_RET
get_resizer "$format" "$resizer" ||
        fail "failed to get a resizer for id '$id'"
resizer=$_RET

debug 1 "resizing $PART on $DISK using $resizer"
"$resizer"

# vi: ts=4 noexpandtab
`
