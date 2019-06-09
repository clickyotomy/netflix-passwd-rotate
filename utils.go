package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/fatih/color"
)

var (
	// For getting login failure reasons.
	netflixLoginUnameInputErr = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/form/div[1]/div[2]`, netflixMnt,
	)
	netflixLoginPwordInputErr = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/form/div[2]/div[2]`, netflixMnt,
	)
	netflixLoginFailErr = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/div/div[2]`, netflixMnt,
	)

	// For getting update failure reasons.
	netflixUpdateOldPwInputErr = `//*[@id="lbl-password"]/div`
	netflixUpdateNewPwInputErr = `//*[@id="lbl-pw_new"]/div`
	netflixUpdateCnfPwInputErr = `//*[@id="lbl-pw_confirm"]/div`

	// Color outputs.
	okColor  = color.New(color.FgGreen).FprintfFunc()
	inpColor = color.New(color.FgWhite).FprintfFunc()
	dbgColor = color.New(color.FgBlue).FprintfFunc()
	infColor = color.New(color.FgMagenta).FprintfFunc()
	wrnColor = color.New(color.FgYellow).FprintfFunc()
	errColor = color.New(color.FgRed).FprintfFunc()
)

// netflixLogin is a wrapper for Netflix login parameters.
type netflixLogin struct {
	username string
	password string

	usernameXpath string
	passwordXpath string

	remXpath string
	subXpath string

	evalXpath string
}

// netflixLogin is a wrapper for Netflix password update parameters.
type netflixPasswordUpdate struct {
	oldPassword string
	newPassword string

	devLogout bool

	oldPasswordXpath    string
	newPasswordXpathNew string
	newPasswordXpathCnf string

	logoutXpath string
	submitXpath string

	evalXpath string
}

// genExecContext creates a new context to start the browser with.
func genExecContext(tmp, exec string) (context.Context, context.CancelFunc) {
	var execAllocOpts = []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.UserDataDir(tmp),
	}

	if exec != "" {
		execAllocOpts = append(execAllocOpts, chromedp.ExecPath(exec))
	}

	return chromedp.NewExecAllocator(context.Background(), execAllocOpts...)
}

// mkTmpDir creates a temporary directory for the user data.
func mkTmpDir(path string) (string, error) {
	return ioutil.TempDir("", path)
}

// loadLoginParams constructs the parameters for the `loginActions' function.
func (n *netflixLogin) loadLoginParams(username, password string) {
	n.username = username
	n.password = password

	n.usernameXpath = `//*[@id="id_userLoginId"]`
	n.passwordXpath = `//*[@id="id_password"]`

	n.remXpath = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/form/div[3]/div/label`, netflixMnt,
	)
	n.subXpath = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/form/button`, netflixMnt,
	)

	n.evalXpath = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/div/div[2]`, netflixMnt,
	)
}

// loadUpdateParams constructs the parameters for the `updateActions' function.
func (n *netflixPasswordUpdate) loadUpdateParams(old, new string, dev bool) {
	n.oldPassword = old
	n.newPassword = new

	n.devLogout = dev

	n.oldPasswordXpath = `//*[@id="password"]`
	n.newPasswordXpathNew = `//*[@id="pw_new"]`
	n.newPasswordXpathCnf = `//*[@id="pw_confirm"]`

	n.logoutXpath = `//*[@id="bxid_signout_devices_signout_devices"]`
	n.submitXpath = fmt.Sprintf(
		`%s/div/div/div[2]/div/div/div/button[1]`, netflixMnt,
	)

	n.evalXpath = fmt.Sprintf(
		`%s/div/div/div[2]/div/div/div[1]/div/div[2]`, netflixMnt,
	)
}

// loginActions returns a set of actions for logging into Netflix.
func loginActions(p *netflixLogin) chromedp.Tasks {
	return chromedp.Tasks{
		// Go to the page, wait for the input boxes to load,
		// and key in the login credentials.
		chromedp.Navigate(netflixPasswordRoute),
		chromedp.WaitVisible(p.usernameXpath),
		chromedp.WaitVisible(p.passwordXpath),
		chromedp.SendKeys(p.usernameXpath, p.username),
		chromedp.SendKeys(p.passwordXpath, p.password),

		// Click on the buttons (submit and remember).
		chromedp.Click(p.remXpath),
		chromedp.Click(p.subXpath),

		// Sleep for a couple of seconds to check login status later.
		chromedp.Sleep(netflixVerifyWait * time.Second),
	}
}

// updateActions returns a set of actions for updating the password.
func updateActions(p *netflixPasswordUpdate) chromedp.Tasks {
	var tasks = chromedp.Tasks{
		// Wait for the input boxes to load,
		// and key in the login credentials.
		chromedp.WaitVisible(p.oldPasswordXpath),
		chromedp.WaitVisible(p.newPasswordXpathNew),
		chromedp.WaitVisible(p.newPasswordXpathCnf),
		chromedp.SendKeys(p.oldPasswordXpath, p.oldPassword),
		chromedp.SendKeys(p.newPasswordXpathNew, p.newPassword),
		chromedp.SendKeys(p.newPasswordXpathCnf, p.newPassword),
	}

	// For logging out of all devices.
	if !p.devLogout {
		tasks = append(tasks, chromedp.Click(p.logoutXpath))
	}

	// Other tasks.
	// Click the submit button.
	tasks = append(tasks, chromedp.Click(p.submitXpath))

	// Sleep for a couple of seconds to check update status later.
	tasks = append(tasks, chromedp.Sleep(netflixVerifyWait*time.Second))

	return tasks
}

// jsEval evaluates a JavaScript expression.
func jsEval(ctx context.Context, expr string) (bool, error) {
	var (
		err  error
		eval bool
	)

	err = chromedp.Run(ctx, chromedp.Evaluate(expr, &eval))
	return eval, err
}

// extractText tries to extract the text from a given selector.
func extractText(ctx context.Context, xPath string) string {
	var (
		err error
		txt string
	)

	err = chromedp.Run(
		ctx,
		chromedp.Evaluate(fmt.Sprintf(netflixEval, xPath, ".innerText"), &txt),
	)
	if err != nil {
		txt = "N/A."
	}

	return txt
}

// getFailureReason gets the reason for failed actions.
func getFailureReason(ctx context.Context, action string) (string, bool) {
	var (
		ok  bool
		sel string

		login = []string{
			netflixLoginUnameInputErr,
			netflixLoginPwordInputErr,
			netflixLoginFailErr,
		}

		update = []string{
			netflixUpdateOldPwInputErr,
			netflixUpdateNewPwInputErr,
			netflixUpdateCnfPwInputErr,
		}
	)

	switch action {
	case "login":
		for _, sel = range login {
			ok, _ = jsEval(ctx, fmt.Sprintf(netflixEval, sel, " !== null"))
			if ok {
				return extractText(ctx, sel), true
			}
		}
	case "update":
		for _, sel = range update {
			ok, _ = jsEval(ctx, fmt.Sprintf(netflixEval, sel, " !== null"))
			if ok {
				return extractText(ctx, sel), true
			}
		}
	default:
		return "Unknown error.", true
	}

	return "", false
}

// exec runs a given set of tasks.
func exec(ctx context.Context, tasks chromedp.Tasks) error {
	return chromedp.Run(ctx, tasks)
}

// exit is a handler function.
func exit(status *int) {
	os.Exit(*status)
}
