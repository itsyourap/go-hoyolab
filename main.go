package main

import (
	"bufio"
	"fmt"
	"hoyolab/act"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tmilewski/goenv"
)

// at *day 1* your got [GSI] x1, [HSR] x1, [HI3] x1
var configExt string = "yaml"
var logExt string = "log"
var configPath string = ""
var logPath string = ""
var logfile *os.File

func init() {
	err := goenv.Load()
	if err != nil {
		log.Println(err)
	}
	IsDev := os.Getenv(act.DEBUG) != ""

	execFilename, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	baseFilename := strings.ReplaceAll(filepath.Base(execFilename), filepath.Ext(execFilename), "")
	dirname := filepath.Dir(execFilename)

	if IsDev {
		dirname, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	}

	configPath = path.Join(dirname, fmt.Sprintf("%s.%s", baseFilename, configExt))
	logPath = path.Join(dirname, fmt.Sprintf("%s.%s", baseFilename, logExt))

	log.SetFlags(log.Lshortfile | log.Ltime)
	if !IsDev {
		log.SetFlags(log.Ldate | log.Ltime)
		f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(f)
	}
}

func main() {
	hoyo := GenerateDefaultConfig()
	if err := hoyo.ReadHoyoConfig(configPath); err != nil {
		log.Fatal(err)
	}
	var notifyMessage []string

	cookiesStr := os.Getenv("HOYOLAB_COOKIES")

	cookies, err := convertStrCookies(cookiesStr)
	if err != nil {
		log.Fatal(err)
	}

	for _, hoyoAct := range hoyo.Daily {
		hoyoAct.SetCookie(cookies)
	}

	for _, profile := range hoyo.Browser {
		var resAcc *act.ActUser
		var err error
		hoyo.Client = resty.New()

		var getDaySign int32 = -1
		var getAward []string
		for i := 0; i < len(hoyo.Daily); i++ {
			hoyoAct := hoyo.Daily[i]
			hoyoAct.UserAgent = profile.UserAgent

			if resAcc == nil {
				resAcc, err = hoyoAct.GetAccountUserInfo(hoyo)
				if err != nil {
					log.Printf("%s::GetUserInfo    : %v", hoyoAct.Label, err)
					continue
				}
				log.Printf("%s::GetUserInfo    : Hi, '%s'", hoyoAct.Label, resAcc.UserInfo.NickName)
			}

			resAward, err := hoyoAct.GetMonthAward(hoyo)
			if err != nil {
				log.Printf("%s::GetMonthAward  : %v", hoyoAct.Label, err)
				continue
			}

			resInfo, err := hoyoAct.GetCheckInInfo(hoyo)
			if err != nil {
				log.Printf("%s::GetCheckInInfo :%v", hoyoAct.Label, err)
				continue
			}

			log.Printf("%s::GetCheckInInfo : Checked in for %d days", hoyoAct.Label, resInfo.TotalSignDay)
			if resInfo.IsSign {
				log.Printf("%s::DailySignIn    : Claimed %s", hoyoAct.Label, resInfo.Today)
				continue
			}

			isRisk, err := hoyoAct.DailySignIn(hoyo)
			if err != nil {
				log.Printf("%s::DailySignIn    : %v", hoyoAct.Label, err)
				continue
			}
			if getDaySign < 0 {
				getDaySign = resInfo.TotalSignDay + 1
			}

			award := resAward.Awards[resInfo.TotalSignDay+1]
			log.Printf("%s::GetMonthAward  : Today's received %s x%d", hoyoAct.Label, award.Name, award.Count)

			if hoyo.Notify.Mini {
				if isRisk {
					getAward = append(getAward, fmt.Sprintf("Challenge captcha (%s)", hoyoAct.Label))
				} else {
					getAward = append(getAward, fmt.Sprintf("*%s x%d* (%s)", award.Name, award.Count, hoyoAct.Label))
				}
			} else {
				if isRisk {
					getAward = append(getAward, fmt.Sprintf("*[%s]* at day %d challenge captcha", hoyoAct.Label, resInfo.TotalSignDay+1))
				} else {
					getAward = append(getAward, fmt.Sprintf("*[%s]* at day %d received %s x%d", hoyoAct.Label, resInfo.TotalSignDay+1, award.Name, award.Count))
				}
			}
		}
		if len(getAward) > 0 {
			if len(hoyo.Browser) > 1 {
				notifyMessage = append(notifyMessage, "\n")
			}

			if hoyo.Notify.Mini {
				notifyMessage = append(notifyMessage, fmt.Sprintf("%s, at day %d your got %s", resAcc.UserInfo.NickName, getDaySign, strings.Join(getAward, ", ")))
			} else {
				notifyMessage = append(notifyMessage, fmt.Sprintf("\nHi, %s Checked in for %d days.\n%s", resAcc.UserInfo.NickName, 1, strings.Join(getAward, "\n")))
			}
		}
	}
}

func GenerateDefaultConfig() *act.Hoyolab {
	// Genshin Impact
	var apiGenshinImpact = &act.DailyHoyolab{
		CookieJar: []*http.Cookie{},
		Label:     "GSI",
		ActID:     "e202102251931481",
		API: act.DailyAPI{
			Endpoint: "https://sg-hk4e-api.hoyolab.com",
			Domain:   "https://hoyolab.com",
			Award:    "/event/sol/home",
			Info:     "/event/sol/info",
			Sign:     "/event/sol/sign",
		},
		Lang:    "en-us",
		Referer: "https://act.hoyolab.com/ys/event/signin-sea-v3/index.html",
	}

	// Honkai StarRail
	var apiHonkaiStarRail = &act.DailyHoyolab{
		CookieJar: []*http.Cookie{},
		Label:     "HSR",
		ActID:     "e202303301540311",
		API: act.DailyAPI{
			Endpoint: "https://sg-public-api.hoyolab.com",
			Domain:   "https://hoyolab.com",
			Award:    "/event/luna/os/home",
			Info:     "/event/luna/os/info",
			Sign:     "/event/luna/os/sign",
		},
		Lang:    "en-us",
		Referer: "https://act.hoyolab.com/bbs/event/signin/hkrpg/index.html",
	}

	// Honkai Impact 3
	var apiHonkaiImpact = &act.DailyHoyolab{
		CookieJar: []*http.Cookie{},
		Label:     "HI3",
		ActID:     "e202110291205111",
		API: act.DailyAPI{
			Endpoint: "https://sg-public-api.hoyolab.com",
			Domain:   "https://hoyolab.com",
			Award:    "/event/mani/home",
			Sign:     "/event/mani/sign",
			Info:     "/event/mani/info",
		},
		Lang:    "en-us",
		Referer: "https://act.hoyolab.com/bbs/event/signin-bh3/index.html",
	}
	return &act.Hoyolab{
		Notify: act.LineNotify{
			LINENotify: "",
			Discord:    "",
			Mini:       true,
		},

		Delay: 150,
		Browser: []act.BrowserProfile{
			{
				Browser:   "chrome",
				Name:      []string{},
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36",
			},
		},
		Daily: []*act.DailyHoyolab{
			apiGenshinImpact,
			apiHonkaiStarRail,
			apiHonkaiImpact,
		},
	}
}

func convertStrCookies(strCookies string) ([]*http.Cookie, error) {
	rawRequest := fmt.Sprintf("GET / HTTP/1.0\r\nCookie: %s\r\n\r\n", strCookies)
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(rawRequest)))

	if err != nil {
		return nil, err
	}
	return req.Cookies(), nil
}
