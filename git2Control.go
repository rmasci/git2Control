package main

/*
|------------------------------------------------------------------------
| This program is free software: you can redistribute it and/or modify  |
| it under the terms of the GNU General Public License as published by  |
| the Free Software Foundation, version 3 of the License.               |
|                                                                       |
| This program is distributed in the hope that it will be useful,       |
| but WITHOUT ANY WARRANTY; without even the implied warranty of        |
| MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the         |
| GNU General Public License for more details.                          |
|                                                                       |
| You should have received a copy of the GNU General Public License     |
| along with this program.  If not, see <http://www.gnu.org/licenses/>. |
|------------------------------------------------------------------------
Written by Richard Masci.
*/

import (
	"encoding/xml"
	"fmt"
	"github.com/spf13/pflag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var searchIP string
var verb, help bool

type xmlParse struct {
	Function xml.Name `xml:"Function"`
	Status   string   `xml:"Status"`
}

func main() {
	var camFunc string
	var connIp, camPar, cmd string
	pflag.StringVarP(&camFunc, "cam", "c", "startCam", "Stop the camera. Default is to start.")
	pflag.StringVarP(&cmd, "cmd", "C", "", "Send camera command. (2001, 20017 etc... see Novatek API.")
	pflag.StringVarP(&camPar, "par", "x", "", "Parameter to send with command")
	pflag.StringVarP(&searchIP, "searchSub", "i", "10.1.1", "First three octets of subnet to search for.")
	pflag.BoolVarP(&verb, "verbose", "v", false, "Verbose")
	pflag.BoolVarP(&help, "help", "h", false, "Help")
	pflag.Parse()
	if help {
		fmt.Printf("Git2Control written by Richard Masci. https://github.com/rmasci/git2Control\n\n")
		fmt.Printf("Commands:startCam\n\tstopCam\n\ttakePic\n\trecStatus\n\n")

		pflag.PrintDefaults()
		os.Exit(0)
	}
	myip := myIpIs()
	cidrIp, fresh := readConf()
	// if no file exists -- the scan starts with the solo's address Camera should be
	// above this address
	if cidrIp == "" {
		cidrIp = myip
	}
	// if the config file is 'fresh' or less than 25 min old -- run with it.
	// most flights last for less than 25 min.
	if fresh {
		connIp = strings.Split(cidrIp, "/")[0]
	} else {
		connIp = findIp(cidrIp)
		if connIp == "" {
			if myip == cidrIp {
				fmt.Println("I can't find the camera")
				os.Exit(1)
			} else {
				connIp = findIp(myip)
				if connIp == "" {
					fmt.Println("I can't find the camera")
					os.Exit(1)
				}
			}
		}
		writeConf(connIp)
	}
	switch camFunc {
	case "startCam":
		fmt.Printf("Starting Camera\n")
		bod, hstat := ctlCamera(connIp, "2001", "par=1")
		log.Printf("Status: %s, %s\n", bod, hstat)
	case "stopCam":
		fmt.Printf("Stopping Camera\n")
		bod, hstat := ctlCamera(connIp, "2001", "par=0")
		log.Printf("Status: %s, %s\n", bod, hstat)
	case "recStatus":
		bod, hstat := ctlCamera(connIp, "2001", "")
		fmt.Printf("Camera Rec Status:%s,%s\n", bod, hstat)
	case "takePic":
		fmt.Printf("Take picture.\n")
		bod, hstat := ctlCamera(connIp, "2017", "")
		log.Printf("Status: %s, %s\n", bod, hstat)
	default:
		var bod, hstat string
		if cmd != "" {
			if camPar != "" {
				fmt.Printf("Running Command: %s with parameter: %s\n")
				bod, hstat = ctlCamera(connIp, cmd, "par="+camPar)
			} else {
				fmt.Printf("Running Command: %s\n")
				bod, hstat = ctlCamera(connIp, cmd, "")

			}
			log.Printf("Status: %s, %s\n", bod, hstat)
		}
	}

}

func errorHandle(err error, str string) {
	if err != nil {
		log.Printf("%s Error: %v\n", str, err)
		os.Exit(1)
	}
}

func myIpIs() string {
	myIps, err := net.InterfaceAddrs()
	errorHandle(err, "Get interface addresses")
	for _, i := range myIps {
		if strings.Contains(i.String(), searchIP) {
			return i.String()
			os.Exit(0)
		}
	}
	return ""
}

func findIp(cidrIp string) string {

	if verb {
		fmt.Printf("CidrIP: %v\n", cidrIp)
	}
	toIp := 254
	tmOut := 100 * time.Millisecond
	myIp := strings.Split(cidrIp, ".")
	lastOct, err := strconv.Atoi(strings.Split(myIp[3], "/")[0])
	errorHandle(err, "Convert octet to int")
	mask := strings.Split(myIp[3], "/")[1]
	if verb {
		fmt.Printf("Last: %v, Mask: %v\n", lastOct, mask)
	}
	if mask == "32" {
		toIp = lastOct
	} else {
		lastOct++
	}
	firstThree := strings.Join(myIp[:3], ".")
	if verb {
		fmt.Println(cidrIp)
		fmt.Println(lastOct)
		fmt.Println(firstThree)
	}

	for i := lastOct; i <= toIp; i++ {
		ipAddr := fmt.Sprintf("%s.%v", firstThree, i)
		tcpAddr := fmt.Sprintf("%v:3333", ipAddr)
		conn, err := net.DialTimeout("tcp4", tcpAddr, tmOut)
		if err == nil {
			fmt.Printf("Found Camera on: %v, sending command.\n", ipAddr)
			conn.Close()
			return ipAddr
		} else {
			if verb {
				fmt.Printf("Not on %v -- %v\n", tcpAddr, err)
			}
		}
	}
	return ""
}

func ctlCamera(connIp, cmd, par string) (cmdStat, hStat string) {
	var xOut xmlParse
	var hGet string
	if par != "" {
		hGet = fmt.Sprintf("http://%v/?custom=1&cmd=%s&%v", connIp, cmd, par)
	} else {
		hGet = fmt.Sprintf("http://%v/?custom=1&cmd=%s", connIp, cmd)
	}
	fmt.Println(hGet)
	res, err := http.Get(hGet)
	errorHandle(err, "HTTP Get")
	o, err := ioutil.ReadAll(res.Body)
	errorHandle(err, "HTTP Read")
	err = xml.Unmarshal(o, xOut)
	cmdStat = xOut.Status
	hStat = res.Status
	res.Body.Close()
	return cmdStat, hStat
}

func readConf() (cidrIp string, fresh bool) {
	fresh = false
	curDir, err := os.Getwd()
	errorHandle(err, "Get Current Dir to write file")
	fileName := fmt.Sprintf("%v/git2IP.txt", curDir)
	if verb {
		fmt.Printf("Reading File: %v\n", fileName)
	}
	retStr, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", fresh
	}
	if verb {
		fmt.Printf("Conn Ip: %v\n", string(retStr))
	}
	// find out if the conf file is fresh
	// Test time.. If < 25 min return w/o scan
	nowTime := time.Now().Unix()
	fileInfo, err := os.Stat("git2IP.txt")
	errorHandle(err, "os.Stat -- couldn't stat file")
	fileTime := fileInfo.ModTime().Unix()
	if nowTime-fileTime < 1500 {
		fresh = true
	}
	return string(retStr), fresh
}

func writeConf(camIp string) {
	curDir, err := os.Getwd()
	camIp = camIp + "/32"
	errorHandle(err, "Get Current Dire to write file")
	fileName := fmt.Sprintf("%v/git2IP.txt", curDir)
	if verb {
		fmt.Printf("Writing file...\n")
	}
	camIpByte := []byte(camIp)
	err = ioutil.WriteFile(fileName, camIpByte, 0644)
	errorHandle(err, "write conf file")
}
