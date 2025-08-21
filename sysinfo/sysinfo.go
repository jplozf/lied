package sysinfo

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"fmt"
	"lied/conf"
	"lied/utils"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type SysInfo struct {
	Variable string
	Value    string
}

// ****************************************************************************
// GetFullReport()
// ****************************************************************************
func GetFullReport() string {
	out := ""

	sTitle, sReport := GetSelfReportAsArray()
	out += "[white]⯈ [red]" + doubleTitle(sTitle) + "[white]\n"
	max := getLongestVariableName(sReport) + 3
	for _, h := range sReport {
		out += "[white]" + h.Variable + strings.Repeat("_", max-len(h.Variable)) + " : [red]" + h.Value + "\n"
	}

	gTitle, gReport := GetGoReportAsArray()
	out += "\n[white]⯈ [red]" + doubleTitle(gTitle) + "[white]\n"
	max = getLongestVariableName(gReport) + 3
	for _, h := range gReport {
		out += "[white]" + h.Variable + strings.Repeat("_", max-len(h.Variable)) + " : [red]" + h.Value + "\n"
	}

	hTitle, hReport := GetHostReportAsArray()
	out += "\n[white]⯈ [red]" + doubleTitle(hTitle) + "[white]\n"
	max = getLongestVariableName(hReport) + 3
	for _, h := range hReport {
		out += "[white]" + h.Variable + strings.Repeat("_", max-len(h.Variable)) + " : [red]" + h.Value + "\n"
	}

	oTitle, oReport := GetOSReportAsArray()
	out += "\n[white]⯈ [red]" + doubleTitle(oTitle) + "[white]\n"
	max = getLongestVariableName(oReport) + 3
	for _, h := range oReport {
		out += "[white]" + h.Variable + strings.Repeat("_", max-len(h.Variable)) + " : [red]" + h.Value + "\n"
	}

	mTitle, mReport := GetMemoryReportAsArray()
	out += "\n[white]⯈ [red]" + doubleTitle(mTitle) + "[white]\n"
	max = getLongestVariableName(mReport) + 3
	for _, h := range mReport {
		out += "[white]" + h.Variable + strings.Repeat("_", max-len(h.Variable)) + " : [red]" + h.Value + "\n"
	}

	lTitle, lReport := GetLoadReportAsArray()
	out += "\n[white]⯈ [red]" + doubleTitle(lTitle) + "[white]\n"
	max = getLongestVariableName(lReport) + 3
	for _, h := range lReport {
		out += "[white]" + h.Variable + strings.Repeat("_", max-len(h.Variable)) + " : [red]" + h.Value + "\n"
	}

	nTitle, nReport := GetNetworkReportAsArray()
	out += "\n[white]⯈ [red]" + doubleTitle(nTitle) + "[white]\n"
	max = getLongestVariableName(nReport) + 3
	for _, h := range nReport {
		out += "[white]" + h.Variable + strings.Repeat("_", max-len(h.Variable)) + " : [red]" + h.Value + "\n"
	}

	eTitle, eReport := GetEnvReportAsArray()
	out += "\n[white]⯈ [red]" + doubleTitle(eTitle) + "[white]\n"
	max = getLongestVariableName(eReport) + 3
	for _, h := range eReport {
		out += "[white]" + h.Variable + strings.Repeat("_", max-len(h.Variable)) + " : [red]" + h.Value + "\n"
	}

	return out
}

// ****************************************************************************
// GetHostReportAsArray()
// ****************************************************************************
func GetHostReportAsArray() (string, []SysInfo) {
	var hReport []SysInfo
	host, _ := sysinfo.Host()
	hReport = append(hReport, SysInfo{"Hostname", host.Info().Hostname})
	hReport = append(hReport, SysInfo{"Architecture", host.Info().Architecture})
	hReport = append(hReport, SysInfo{"Kernel Version", host.Info().KernelVersion})
	hReport = append(hReport, SysInfo{"Time Zone", host.Info().Timezone})
	hReport = append(hReport, SysInfo{"Unique ID", host.Info().UniqueID})
	hReport = append(hReport, SysInfo{"Boot Time", host.Info().BootTime.String()})

	return "HOST", hReport
}

// ****************************************************************************
// GetNetworkReportAsArray()
// ****************************************************************************
func GetNetworkReportAsArray() (string, []SysInfo) {
	var nReport []SysInfo
	host, _ := sysinfo.Host()
	for i, s := range host.Info().IPs {
		nReport = append(nReport, SysInfo{"IP address #" + strconv.Itoa(i), s})
	}
	for i, s := range host.Info().MACs {
		nReport = append(nReport, SysInfo{"MAC address #" + strconv.Itoa(i), s})
	}

	return "NETWORK", nReport
}

// ****************************************************************************
// GetOSReportAsArray()
// ****************************************************************************
func GetOSReportAsArray() (string, []SysInfo) {
	var oReport []SysInfo
	host, _ := sysinfo.Host()
	oReport = append(oReport, SysInfo{"Name", host.Info().OS.Name})
	oReport = append(oReport, SysInfo{"Version", host.Info().OS.Version})
	oReport = append(oReport, SysInfo{"Codename", host.Info().OS.Codename})
	oReport = append(oReport, SysInfo{"Type", host.Info().OS.Type})
	oReport = append(oReport, SysInfo{"Family", host.Info().OS.Family})
	oReport = append(oReport, SysInfo{"Platform", host.Info().OS.Platform})

	return "OS", oReport
}

// ****************************************************************************
// GetMemoryReportAsArray()
// ****************************************************************************
func GetMemoryReportAsArray() (string, []SysInfo) {
	var mReport []SysInfo
	host, _ := sysinfo.Host()
	memo, _ := host.Memory()
	mReport = append(mReport, SysInfo{"Total", HumanSize(memo.Total)})
	mReport = append(mReport, SysInfo{"Used", HumanSize(memo.Used)})
	mReport = append(mReport, SysInfo{"Available", HumanSize(memo.Available)})
	mReport = append(mReport, SysInfo{"Free", HumanSize(memo.Free)})
	mReport = append(mReport, SysInfo{"Virtual Total", HumanSize(memo.VirtualTotal)})
	mReport = append(mReport, SysInfo{"Virtual Used", HumanSize(memo.VirtualUsed)})
	mReport = append(mReport, SysInfo{"Virtual Free", HumanSize(memo.VirtualFree)})

	return "MEMORY", mReport
}

// ****************************************************************************
// GetSelfReportAsArray()
// ****************************************************************************
func GetSelfReportAsArray() (string, []SysInfo) {
	var sReport []SysInfo
	sReport = append(sReport, SysInfo{"App. String", conf.APP_STRING})
	sReport = append(sReport, SysInfo{"App. Version", conf.Version})
	sReport = append(sReport, SysInfo{"App. URL", conf.APP_URL})
	self, _ := sysinfo.Self()
	sReport = append(sReport, SysInfo{"PID", strconv.Itoa(self.PID())})
	return strings.ToUpper(conf.APP_NAME), sReport
}

// ****************************************************************************
// GetLoadReportAsArray()
// ****************************************************************************
func GetLoadReportAsArray() (string, []SysInfo) {
	var lReport []SysInfo
	host, _ := sysinfo.Host()
	load, _ := host.(types.LoadAverage)
	l, _ := load.LoadAverage()
	lReport = append(lReport, SysInfo{"One Minute", fmt.Sprintf("%v", l.One)})
	lReport = append(lReport, SysInfo{"Five Minutes", fmt.Sprintf("%v", l.Five)})
	lReport = append(lReport, SysInfo{"Fifteen Minutes", fmt.Sprintf("%v", l.Fifteen)})

	return "LOAD AVERAGE", lReport
}

// ****************************************************************************
// GetEnvReportAsArray()
// ****************************************************************************
func GetEnvReportAsArray() (string, []SysInfo) {
	var eReport []SysInfo
	self, _ := sysinfo.Self()
	myProcess, _ := sysinfo.Process(self.PID())
	env, _ := myProcess.(types.Environment)
	e, _ := env.Environment()

	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		eReport = append(eReport, SysInfo{k, e[k]})
	}

	return "ENVIRONMENT", eReport
}

// ****************************************************************************
// GetGoReportAsArray()
// ****************************************************************************
func GetGoReportAsArray() (string, []SysInfo) {
	var gReport []SysInfo
	ggo := sysinfo.Go()
	gReport = append(gReport, SysInfo{"OS", ggo.OS})
	gReport = append(gReport, SysInfo{"Architecture", ggo.Arch})
	gReport = append(gReport, SysInfo{"Version", ggo.Version})
	gReport = append(gReport, SysInfo{"Max Procs", strconv.Itoa(ggo.MaxProcs)})

	return "GO RUNTIME", gReport
}

// ****************************************************************************
// getLongestVariableName()
// ****************************************************************************
func getLongestVariableName(r []SysInfo) int {
	l := 0
	for _, n := range r {
		if len(n.Variable) > l {
			l = len(n.Variable)
		}
	}
	return l
}

// ****************************************************************************
// doubleTitle()
// ****************************************************************************
func doubleTitle(t string) string {
	out := ""
	for _, char := range t {
		out = out + string(char) + " "
	}
	return strings.TrimSpace(out)
}

// ****************************************************************************
// HumanSize()
// ****************************************************************************
func HumanSize(size uint64) string {
	var suffixes [5]string
	fSize := float64(size)
	if size == 0 {
		return "0 B"
	} else {
		suffixes[0] = "B"
		suffixes[1] = "KB"
		suffixes[2] = "MB"
		suffixes[3] = "GB"
		suffixes[4] = "TB"

		base := math.Log(fSize) / math.Log(1024)
		getSize := utils.Round(math.Pow(1024, base-math.Floor(base)), .5, 2)
		getSuffix := suffixes[int(math.Floor(base))]
		return strconv.FormatFloat(getSize, 'f', -1, 64) + " " + string(getSuffix)
	}
}

// ****************************************************************************
// GetHTMLReport()
// ****************************************************************************
func GetHTMLReport() string {
	out := "<!DOCTYPE html\n"
	out += "PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\"\n"
	out += "\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">\n"
	out += "<html>\n"
	out += "<head>\n"
	out += `
<style>
table {
  border-collapse: collapse;
}

th {
  background: #ccc;
}

th, td {
  border: 1px solid #ccc;
  padding: 8px;
}

tr:nth-child(even) {
  background: #F0FFFF;
}

tr:hover {
  background: #ADD8E6;
}
  
h1 {
  color: #0085c3;
  font-family: Helvetica Neue, Helvetica, Arial, sans-serif;
  font-size: 33px;
  font-weight: 600;
  line-height: 30px;
  margin: 40px auto 25px;
  text-align: center;
}

h1 span {
  border-bottom: 1px solid #0085c3;
  padding-bottom: 10px;
}

h2 {
  background-color: #0085c3;
  padding: 10px;
  color: #fff;
  border: none;
  border-radius: 5px;
}

.subtitle {font-size: 0.9rem; color: #767676;}

* {
 font-family: monospace, monospace;
}

</style>
  `
	out += "<title>System Information</title>\n"
	out += "</head>\n"
	out += "<body>\n"
	out += "<h1><span>System Information</span></h1>\n"
	out += "<br><span class=\"subtitle\">Generated : " + time.Now().Format("20060102-150405") + "</span>\n"

	sTitle, sReport := GetSelfReportAsArray()
	out += "<h2>⯈ " + sTitle + "</h2>\n"
	out += "<table>\n"
	for _, h := range sReport {
		if strings.HasPrefix(h.Value, "http") {
			out += "<tr><td>" + h.Variable + "</td><td><a href=\"" + h.Value + "\" target=\"_blank\"><b>" + h.Value + "</b></a></td></tr>\n"
		} else {
			out += "<tr><td>" + h.Variable + "</td><td><b>" + h.Value + "</b></td></tr>\n"
		}
	}
	out += "</table>\n"

	gTitle, gReport := GetGoReportAsArray()
	out += "<h2>⯈ " + gTitle + "</h2>\n"
	out += "<table>\n"
	for _, h := range gReport {
		out += "<tr><td>" + h.Variable + "</td><td><b>" + h.Value + "</b></td></tr>\n"
	}
	out += "</table>\n"

	hTitle, hReport := GetHostReportAsArray()
	out += "<h2>⯈ " + hTitle + "</h2>\n"
	out += "<table>\n"
	for _, h := range hReport {
		out += "<tr><td>" + h.Variable + "</td><td><b>" + h.Value + "</b></td></tr>\n"
	}
	out += "</table>\n"

	oTitle, oReport := GetOSReportAsArray()
	out += "<h2>⯈ " + oTitle + "</h2>\n"
	out += "<table>\n"
	for _, h := range oReport {
		out += "<tr><td>" + h.Variable + "</td><td><b>" + h.Value + "</b></td></tr>\n"
	}
	out += "</table>\n"

	mTitle, mReport := GetMemoryReportAsArray()
	out += "<h2>⯈ " + mTitle + "</h2>\n"
	out += "<table>\n"
	for _, h := range mReport {
		out += "<tr><td>" + h.Variable + "</td><td><b>" + h.Value + "</b></td></tr>\n"
	}
	out += "</table>\n"

	lTitle, lReport := GetLoadReportAsArray()
	out += "<h2>⯈ " + lTitle + "</h2>\n"
	out += "<table>\n"
	for _, h := range lReport {
		out += "<tr><td>" + h.Variable + "</td><td><b>" + h.Value + "</b></td></tr>\n"
	}
	out += "</table>\n"

	nTitle, nReport := GetNetworkReportAsArray()
	out += "<h2>⯈ " + nTitle + "</h2>\n"
	out += "<table>\n"
	for _, h := range nReport {
		out += "<tr><td>" + h.Variable + "</td><td><b>" + h.Value + "</b></td></tr>\n"
	}
	out += "</table>\n"

	eTitle, eReport := GetEnvReportAsArray()
	out += "<h2>⯈ " + eTitle + "</h2>\n"
	out += "<table>\n"
	for _, h := range eReport {
		if strings.HasPrefix(h.Value, "http") {
			out += "<tr><td>" + h.Variable + "</td><td><a href=\"" + h.Value + "\" target=\"_blank\"><b>" + h.Value + "</b></a></td></tr>\n"
		} else {
			out += "<tr><td>" + h.Variable + "</td><td><b>" + h.Value + "</b></td></tr>\n"
		}
	}
	out += "</table>\n"

	out += "</body>\n"
	out += "</html>\n"

	return out
}
