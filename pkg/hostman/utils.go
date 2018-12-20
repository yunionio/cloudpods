package hostman

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
)

// timer utils

func AddTimeout(second time.Duration, callback func()) {
	go func() {
		<-time.NewTimer(second).C
		callback()
	}()
}

func CommandWithTimeout(timeout int, cmds ...string) *exec.Cmd {
	if timeout > 0 {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", timeout)}, cmds...)
	}
	return exec.Command(cmds[0], cmds[1:]...)
}

// file utils

// TODO: test
func Cleandir(sPath string, keepdir bool) error {
	if f, _ := os.Lstat(sPath); f == nil || f.Mode()&os.ModeSymlink == os.ModeSymlink {
		return nil
	}
	files, _ := ioutil.ReadDir(sPath)
	for _, file := range files {
		fp := path.Join(sPath, file.Name())
		if f, _ := os.Lstat(fp); f.Mode()&os.ModeSymlink == os.ModeSymlink {
			if !keepdir {
				if err := os.Remove(fp); err != nil {
					return err
				}
			}
		} else if f.IsDir() {
			Cleandir(fp, keepdir)
			if !keepdir {
				if err := os.Remove(fp); err != nil {
					return err
				}
			}
		} else {
			if err := os.Remove(fp); err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO: test
func Zerofiles(sPath string) error {
	f, err := os.Lstat(sPath)
	switch {
	case err != nil:
		return err
	case f.Mode()&os.ModeSymlink == os.ModeSymlink:
		// islink
		return nil
	case f.Mode().IsRegular():
		return FilePutContents(sPath, "", false)
	case f.Mode().IsDir():
		files, err := ioutil.ReadDir(sPath)
		if err != nil {
			return err
		}
		for _, file := range files {
			if file.Mode()&os.ModeSymlink == os.ModeSymlink {
				continue
			} else if file.Mode().IsRegular() {
				if err := FilePutContents(path.Join(sPath, file.Name()), "", false); err != nil {
					return err
				}
			} else if file.Mode().IsDir() {
				return Zerofiles(path.Join(sPath, file.Name()))
			}
		}
	}
	return nil
}

func FilePutContents(filename string, content string, modAppend bool) error {
	var mode = os.O_WRONLY | os.O_CREATE
	if modAppend {
		mode = mode | os.O_APPEND
	}
	fd, err := os.OpenFile(filename, mode, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.WriteString(content)
	return err
}

func IsBlockDevMounted(dev string) bool {
	devPath := "/dev/" + dev
	mounts, err := exec.Command("mount").Output()
	if err != nil {
		return false
	}
	for _, s := range strings.Split(string(mounts), "\n") {
		if strings.HasPrefix(s, devPath) {
			return true
		}
	}
	return false
}

func IsBlockDeviceUsed(dev string) bool {
	if strings.HasPrefix(dev, "/dev/") {
		dev = dev[strings.LastIndex(dev, "/")+1:]
	}
	devStr := fmt.Sprint(" %s\n", dev)
	devs, _ := exec.Command("cat", "/proc/partitions").Output()
	if idx := strings.Index(string(devs), devStr); idx > 0 {
		return false
	}
	return true
}

func ChangeAllBlkdevsParams(params map[string]string) {
	if _, err := os.Stat("/sys/block"); !os.IsNotExist(err) {
		blockDevs, err := ioutil.ReadDir("/sys/block")
		if err != nil {
			log.Errorln(err)
			return
		}
		for _, b := range blockDevs {
			if IsBlockDevMounted(b.Name()) {
				for k, v := range params {
					ChangeBlkdevParameter(b.Name(), k, v)
				}
			}
		}
	}
}

func ChangeBlkdevParameter(dev, key, value string) {
	p := path.Join("/sys/block", dev, key)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		err = FilePutContents(p, value, false)
		if err != nil {
			log.Errorf("Fail to set %s of %s to %s:%s", key, dev, value, err)
		}
		log.Infof("Set %s of %s to %s", key, dev, value)
	}
}

/*
func PathNotExists(path string) bool {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return true
    }
    return false
}

func PathExists(path string) bool {
    if _, err := os.Stat(path); !os.IsNotExist(err) {
        return true
    }
    return false
}
*/

func FileGetContents(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GetFsFormat(diskPath string) string {
	ret, err := exec.Command("blkid", "-o", "value", "-s", "TYPE", diskPath).Output()
	if err != nil {
		return ""
	}
	var res string
	for _, line := range strings.Split(string(ret), "\n") {
		res += line
	}
	return res
}

func CleanFailedMountpoints() {
	var mtfile = "/etc/mtab"
	if _, err := os.Stat(mtfile); os.IsNotExist(err) {
		mtfile = "/proc/mounts"
	}
	f, err := os.Open(mtfile)
	if err != nil {
		log.Errorf("CleanFailedMountpoints error: %s", err)
	}
	reader := bufio.NewReader(f)
	line, _, err := reader.ReadLine()
	for err != nil {
		m := strings.Split(string(line), " ")
		if len(m) > 1 {
			mp := m[1]
			if _, err := os.Stat(mp); os.IsNotExist(err) {
				log.Warningf("Mount point %s not exists", mp)
			}
			exec.Command("umount", mp).Run()
		}
	}
}

type HostsFile map[string][]string

func (hf HostsFile) Parse(content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		data := regexp.MustCompile(`\s+`).Split(line, -1)
		for len(data) > 0 && data[len(data)-1] == "" {
			data = data[:len(data)-1]
		}
		if len(data) > 1 {
			hf[data[0]] = data[1:]
		}
	}
}

func (hf HostsFile) Add(name string, value ...string) {
	hf[name] = value
}

func (hf HostsFile) String() string {
	var ret = ""
	for k, v := range hf {
		if len(v) > 0 {
			ret += fmt.Sprintf("%s\t%s\n", k, strings.Join(v, "\t"))
		}
	}
	return ret
}

//net utils
var PSEUDO_VIP = "169.254.169.231"
var MASKS = []string{"0", "128", "192", "224", "240", "248", "252", "254", "255"}

func GetMainNic(nics []jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var mainIp netutils.IPV4Addr
	var mainNic jsonutils.JSONObject
	for _, n := range nics {
		if n.Contains("gateway") {
			ip, _ := n.GetString("ip")
			ipInt, err := netutils.NewIPV4Addr(ip)
			if err != nil {
				return nil, err
			}
			if mainIp > 0 {
				mainIp = ipInt
				mainNic = n
			} else if !netutils.IsPrivate(ipInt) && netutils.IsPrivate(mainIp) {
				mainIp = ipInt
				mainNic = n
			}
		}
	}
	return mainNic, nil
}

func Netlen2Mask(netmasklen int64) string {
	var mask = ""
	var segCnt = 0
	for netmasklen > 0 {
		var m string
		if netmasklen > 8 {
			m = MASKS[8]
			netmasklen -= 8
		} else {
			m = MASKS[netmasklen]
		}
		if mask != "" {
			mask += "."
		}
		mask += m
		segCnt += 1
	}
	for i := 0; i < (4 - segCnt); i++ {
		if mask != "" {
			mask += "."
		}
		mask += "0"
	}
	return mask
}

func addRoute(routes *[][]string, net, gw string) {
	for _, rt := range *routes {
		if rt[0] == net {
			return
		}
	}
	*routes = append(*routes, []string{net, gw})
}


    // if len(nic.Routes) != 0 {
    //     for _, nicRoute := range nic.Routes {
    //         tRoute, err := parseNicRoute(nicRoute)
    //         if err != nil {
    //             log.Errorf("Parse route %v error: %v", nicRoute, err)
    //             continue
    //         }
    //         routes = append(routes, tRoute)
    //     }
    // }


func parseNicRoute(route Route) (*types.Route, error) {
    if len(route) != 2 {
        return nil, fmt.Errorf("Invalid route format: %v", route)
    }
    _, dstNet, err := net.ParseCIDR(route[0])
    if err != nil {
        return nil, err
    }
    gwIP := net.ParseIP(route[1])
    return &types.Route{
        Dst: *dstNet,
        GW:  gwIP,
    }, nil
}



func extendRoutes(routes *[][]string, nicRoutes []jsonutils.JSONObject) error {
	for i := 0; i < len(nicRoutes); i++ {
		rt, ok := nicRoutes[i].(*jsonutils.JSONArray)
		if !ok {
			return fmt.Errorf("Nic routes format error")
		}
		// 写到这里了
		rts := rt.GetStringArray()
		if len(rts) < 2 {
			return fmt.Errorf("Nic routes count error")
		}
		addRoute(routes, rts[0], rts[1])
	}
	return nil
}

func AddNicRoutes(routes *[][]string, nic jsonutils.JSONObject, mainIp string, nicCnt int) {
	var nicDesc = new(SNic)
	nic.Unmarshal(nicDesc)
	ip, _ := nic.GetString("ip")
	if mainIp == ip {
		return
	}
	if nic.Contains("routes") {
		nicRoutes, _ := nic.GetArray("routes")
		extendRoutes(routes, nicRoutes)
	} else if nic.Contains("gateway") && 
}
