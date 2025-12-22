#!/bin/bash
# 排查 eBPF cgroup2 device allow 规则的脚本
# Usage: ./debug_ebpf_cgroup_device.sh <cgroup_path>
# Example: ./debug_ebpf_cgroup_device.sh /sys/fs/cgroup/pod123/container456

set -e

CGROUP_PATH="$1"
if [ -z "$CGROUP_PATH" ]; then
    echo "Usage: $0 <cgroup_path>"
    echo "Example: $0 /sys/fs/cgroup/pod123/container456"
    exit 1
fi

echo "=== eBPF Cgroup2 Device Allow 规则排查 ==="
echo ""

# 1. 检查 cgroup 路径是否存在
echo "1. 检查 cgroup 路径:"
if [ ! -d "$CGROUP_PATH" ]; then
    echo "   [ERROR] Cgroup 路径不存在: $CGROUP_PATH"
    exit 1
else
    echo "   [OK] Cgroup 路径存在: $CGROUP_PATH"
fi

# 2. 检查是否为 cgroup2
echo ""
echo "2. 检查 cgroup 版本:"
if [ -f "$CGROUP_PATH/cgroup.procs" ]; then
    echo "   [OK] 这是 cgroup2 (存在 cgroup.procs)"
else
    echo "   [ERROR] 这不是有效的 cgroup2 目录"
    exit 1
fi

# 3. 检查内核版本
echo ""
echo "3. 检查内核版本:"
KERNEL_VERSION=$(uname -r)
KERNEL_MAJOR=$(echo $KERNEL_VERSION | cut -d. -f1)
KERNEL_MINOR=$(echo $KERNEL_VERSION | cut -d. -f2)
if [ "$KERNEL_MAJOR" -gt 4 ] || ([ "$KERNEL_MAJOR" -eq 4 ] && [ "$KERNEL_MINOR" -ge 15 ]); then
    echo "   [OK] 内核版本: $KERNEL_VERSION (>= 4.15)"
else
    echo "   [WARN] 内核版本: $KERNEL_VERSION (需要 >= 4.15)"
fi

# 4. 检查 eBPF 支持
echo ""
echo "4. 检查 eBPF 支持:"
if [ -d "/sys/fs/bpf" ]; then
    echo "   [OK] /sys/fs/bpf 存在"
else
    echo "   [WARN] /sys/fs/bpf 不存在 (可能正常，如果不使用 pinning)"
fi

# 检查 eBPF 相关的内核配置
if [ -f /proc/config.gz ]; then
    if zcat /proc/config.gz 2>/dev/null | grep -q "CONFIG_BPF=y"; then
        echo "   [OK] CONFIG_BPF 已启用"
    else
        echo "   [WARN] CONFIG_BPF 可能未启用"
    fi
    if zcat /proc/config.gz 2>/dev/null | grep -q "CONFIG_CGROUP_BPF=y"; then
        echo "   [OK] CONFIG_CGROUP_BPF 已启用"
    else
        echo "   [WARN] CONFIG_CGROUP_BPF 可能未启用"
    fi
elif [ -f /boot/config-$(uname -r) ]; then
    if grep -q "CONFIG_BPF=y" /boot/config-$(uname -r); then
        echo "   [OK] CONFIG_BPF 已启用"
    else
        echo "   [WARN] CONFIG_BPF 可能未启用"
    fi
    if grep -q "CONFIG_CGROUP_BPF=y" /boot/config-$(uname -r); then
        echo "   [OK] CONFIG_CGROUP_BPF 已启用"
    else
        echo "   [WARN] CONFIG_CGROUP_BPF 可能未启用"
    fi
else
    echo "   [INFO] 无法检查内核配置 (需要 /proc/config.gz 或 /boot/config-*)"
fi

# 5. 检查 bpftool 是否可用
echo ""
echo "5. 检查 bpftool:"
if command -v bpftool &> /dev/null; then
    echo "   [OK] bpftool 可用"
    echo ""
    echo "   查看附加到该 cgroup 的 eBPF 程序:"
    bpftool cgroup tree "$CGROUP_PATH" 2>/dev/null || echo "   [ERROR] 无法查询 eBPF 程序"
    echo ""
    echo "   查看所有 cgroup device 类型的 eBPF 程序:"
    bpftool prog list | grep -i device || echo "   [INFO] 未找到 device 类型的 eBPF 程序"
else
    echo "   [WARN] bpftool 不可用，无法直接查看附加的 eBPF 程序"
    echo "   可以安装: apt-get install linux-tools-generic 或 yum install bpftool"
fi

# 6. 检查权限
echo ""
echo "6. 检查权限:"
if [ -r "$CGROUP_PATH" ]; then
    echo "   [OK] 有读权限"
else
    echo "   [ERROR] 无读权限"
fi
if [ -w "$CGROUP_PATH" ]; then
    echo "   [OK] 有写权限"
else
    echo "   [WARN] 无写权限 (可能正常)"
fi

# 7. 检查 memlock 限制
echo ""
echo "7. 检查 memlock 限制:"
MEMLOCK=$(ulimit -l 2>/dev/null || echo "unknown")
if [ "$MEMLOCK" = "unlimited" ] || [ "$MEMLOCK" = "unknown" ]; then
    echo "   [OK] memlock 限制: $MEMLOCK"
elif [ "$MEMLOCK" -gt 65536 ] 2>/dev/null; then
    echo "   [OK] memlock 限制: $MEMLOCK"
else
    echo "   [WARN] memlock 限制可能太小: $MEMLOCK (建议 unlimited 或 > 65536)"
    echo "   可以通过 'ulimit -l unlimited' 设置"
fi

# 8. 检查是否有进程在该 cgroup 中
echo ""
echo "8. 检查 cgroup 中的进程:"
if [ -f "$CGROUP_PATH/cgroup.procs" ]; then
    PROCS=$(cat "$CGROUP_PATH/cgroup.procs" 2>/dev/null | wc -l)
    if [ "$PROCS" -gt 0 ]; then
        echo "   [INFO] 有 $PROCS 个进程在该 cgroup 中"
        echo "   进程列表:"
        cat "$CGROUP_PATH/cgroup.procs" | head -5 | while read pid; do
            if [ -n "$pid" ]; then
                echo "     PID $pid: $(ps -p $pid -o comm= 2>/dev/null || echo 'unknown')"
            fi
        done
    else
        echo "   [INFO] 当前没有进程在该 cgroup 中"
        echo "   [NOTE] eBPF 规则会在进程加入 cgroup 后生效"
    fi
fi

# 9. 检查系统日志中的 eBPF 相关错误
echo ""
echo "9. 检查系统日志中的 eBPF 错误:"
if command -v journalctl &> /dev/null; then
    echo "   最近的 eBPF 相关错误 (最近1小时):"
    journalctl -k --since "1 hour ago" 2>/dev/null | grep -i "bpf\|ebpf" | tail -5 || echo "   未找到相关错误"
elif [ -f /var/log/kern.log ]; then
    echo "   最近的 eBPF 相关错误:"
    grep -i "bpf\|ebpf" /var/log/kern.log 2>/dev/null | tail -5 || echo "   未找到相关错误"
elif [ -f /var/log/messages ]; then
    echo "   最近的 eBPF 相关错误:"
    grep -i "bpf\|ebpf" /var/log/messages 2>/dev/null | tail -5 || echo "   未找到相关错误"
else
    echo "   [INFO] 无法访问系统日志"
fi

# 10. 检查 dmesg 中的 eBPF 相关消息
echo ""
echo "10. 检查 dmesg 中的 eBPF 消息:"
if dmesg 2>/dev/null | grep -i "bpf\|ebpf" | tail -5; then
    echo ""
else
    echo "   未找到相关消息"
fi

# 11. 检查 cgroup 的控制器
echo ""
echo "11. 检查 cgroup 控制器:"
if [ -f "$CGROUP_PATH/cgroup.controllers" ]; then
    CONTROLLERS=$(cat "$CGROUP_PATH/cgroup.controllers" 2>/dev/null)
    echo "   可用控制器: $CONTROLLERS"
    if echo "$CONTROLLERS" | grep -q "device"; then
        echo "   [INFO] device 控制器可用 (cgroup v1)"
    else
        echo "   [INFO] device 控制器不可用 (cgroup v2 使用 eBPF，不需要 device 控制器)"
    fi
fi

echo ""
echo "=== 排查完成 ==="
echo ""
echo "如果 eBPF 程序附加成功但仍不生效，可能的原因:"
echo "1. 进程还没有加入到该 cgroup"
echo "2. 设备规则格式不正确"
echo "3. 有其他 eBPF 程序覆盖了规则"
echo "4. 内核版本或配置问题"
echo "5. 权限不足"
echo ""
echo "排查建议:"
echo "- 使用 'bpftool cgroup tree <path>' 查看附加的程序"
echo "- 检查应用日志中的 [SetDevicesAllow] 相关日志"
echo "- 确保进程已加入到正确的 cgroup"
echo "- 测试设备访问以验证规则是否生效"
