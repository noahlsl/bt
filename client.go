package bt

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Client 每个 Client 对象对应一个宝塔面板 先实例化再调用接口
type Client struct {
	BTAddress string         // 目标宝塔面板地址 eg.http://10.0.0.14:8888 结尾不要有斜杠
	BTKey     string         // API Key 还需要添加 IP 白名单
	cookies   []*http.Cookie // 根据文档建议保存每次返回的 cookies 来提高效率
	Timeout   time.Duration
}

// NewClient 填入两个参数来实例化 Client 对象
func NewClient(address string, key string, timeout ...time.Duration) *Client {
	ret := &Client{
		BTAddress: address,
		BTKey:     key,
	}
	if len(timeout) > 0 && timeout[0] != 0 {
		ret.Timeout = timeout[0]
	}
	return ret
}

func (c *Client) btAPI(data map[string][]string, endpoint string) ([]byte, error) {
	requestURL, err := url.Parse(c.BTAddress + endpoint)
	if err != nil {
		panic(err)
	}
	nowTime := strconv.FormatInt(time.Now().Unix(), 10)
	requestToken, requestTime := MD5(nowTime+MD5(c.BTKey)), nowTime
	body := url.Values{
		"request_token": {requestToken},
		"request_time":  {requestTime},
	}
	for k, v := range data {
		body[k] = v
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	client := &http.Client{
		Jar:     jar,
		Timeout: c.Timeout,
	}
	if len(c.cookies) != 0 {
		client.Jar.SetCookies(requestURL, c.cookies)
	}
	resp, err := client.PostForm(requestURL.String(), body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, errors.New(resp.Status)
	}
	// 保存每次返回的 cookies
	c.cookies = resp.Cookies()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

// Deprecated: Used only for debug
// 执行无封装 API 调用
func (c *Client) Raw(data map[string][]string, endpoint string) ([]byte, error) {
	return c.btAPI(data, endpoint)
}

// GetNetWork 获取实时状态信息(CPU、内存、网络、负载)
func (c *Client) GetNetWork() (NetWork, error) {
	resp, err := c.btAPI(map[string][]string{}, "/system?action=GetNetWork")
	if err != nil {
		return NetWork{}, err
	}
	var dec NetWork
	if err := json.Unmarshal(resp, &dec); err != nil {
		return NetWork{}, err
	}
	return dec, nil
}

// GetSystemTotal 获取系统基础统计
func (c *Client) GetSystemTotal() (SystemTotal, error) {
	resp, err := c.btAPI(map[string][]string{}, "/system?action=GetSystemTotal")
	if err != nil {
		return SystemTotal{}, err
	}
	var dec SystemTotal
	if err := json.Unmarshal(resp, &dec); err != nil {
		return SystemTotal{}, err
	}
	return dec, nil
}

// GetDiskInfo 获取磁盘分区信息
func (c *Client) GetDiskInfo() (DiskInfo, error) {
	resp, err := c.btAPI(map[string][]string{}, "/system?action=GetDiskInfo")
	if err != nil {
		return DiskInfo{}, err
	}
	var dec DiskInfo
	if err := json.Unmarshal(resp, &dec); err != nil {
		return DiskInfo{}, err
	}
	return dec, nil
}

// GetTaskCount 检查是否有安装任务
func (c *Client) GetTaskCount() int {
	resp, err := c.btAPI(map[string][]string{}, "/ajax?action=GetTaskCount")
	if err != nil {
		return 0
	}
	dec, err := strconv.Atoi(string(resp))
	if err != nil {
		return 0
	}
	return dec
}

// GetPHPVersion 获取已安装的 PHP 版本列表
func (c *Client) GetPHPVersion() (PHPVersions, error) {
	resp, err := c.btAPI(map[string][]string{}, "/site?action=GetPHPVersion")
	if err != nil {
		return PHPVersions{}, err
	}
	var dec PHPVersions
	if err := json.Unmarshal(resp, &dec); err != nil {
		return PHPVersions{}, err
	}
	return dec, nil
}

// GetUpdateStatus 检查面板更新
func (c *Client) GetUpdateStatus(check bool, force bool) (UpdateStatus, error) {
	data := map[string][]string{
		"check": {strconv.FormatBool(check)},
		"force": {strconv.FormatBool(force)},
	}
	resp, err := c.btAPI(data, "/ajax?action=UpdatePanel")
	if err != nil {
		return UpdateStatus{}, err
	}
	var dec UpdateStatus
	if err := json.Unmarshal(resp, &dec); err != nil {
		return UpdateStatus{}, err
	}
	return dec, nil
}

// GetSites 获取网站列表
func (c *Client) GetSites(params *ReqSites) (RespSites, error) {
	data := map[string][]string{
		"p":      {strconv.FormatInt(params.P, 10)},
		"limit":  {strconv.FormatInt(params.Limit, 10)},
		"type":   {strconv.FormatInt(params.Type, 10)},
		"order":  {params.Order},
		"tojs":   {params.ToJS},
		"search": {params.Search},
	}
	resp, err := c.btAPI(data, "/data?action=getData&table=sites")
	if err != nil {
		return RespSites{}, err
	}
	var dec RespSites
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespSites{}, err
	}
	return dec, nil
}

// AddSite 创建网站
func (c *Client) AddSite(params *ReqAddSite) (RespAddSite, error) {
	webname, err := json.Marshal(params.WebName)
	// fmt.Println(string(webname))
	if err != nil {
		return RespAddSite{}, err
	}
	data := map[string][]string{
		"webname":      {string(webname)},
		"path":         {params.Path},
		"type_id":      {strconv.FormatInt(params.TypeID, 10)},
		"type":         {params.Type},
		"version":      {strconv.FormatInt(params.Version, 10)},
		"port":         {strconv.FormatInt(params.Port, 10)},
		"ps":           {params.PS},
		"ftp":          {strconv.FormatBool(params.FTP)},
		"ftp_username": {params.FTPUserName},
		"ftp_password": {params.FTPPassword},
		"sql":          {strconv.FormatBool(params.SQL)},
		"codeing":      {params.Codeing},
		"datauser":     {params.DataUser},
		"datapassword": {params.DataPassword},
	}
	resp, err := c.btAPI(data, "/site?action=AddSite")
	if err != nil {
		return RespAddSite{}, err
	}
	var dec RespAddSite
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespAddSite{}, err
	}
	return dec, nil
}

// DeleteSite 删除网站
func (c *Client) DeleteSite(params *ReqDeleteSite) (RespMSG, error) {
	data := map[string][]string{
		"id":      {strconv.FormatInt(params.ID, 10)},
		"webname": {params.WebName},
	}
	if params.FTP {
		data["ftp"] = []string{"1"}
	}
	if params.Database {
		data["database"] = []string{"1"}
	}
	if params.Path {
		data["path"] = []string{"1"}
	}
	resp, _ := c.btAPI(data, "/site?action=DeleteSite")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// StopSite 停止网站
func (c *Client) StopSite(id int64, name string) (RespMSG, error) {
	data := map[string][]string{
		"id":   {strconv.FormatInt(id, 10)},
		"name": {name},
	}
	resp, _ := c.btAPI(data, "/site?action=SiteStop")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// StartSite 启动网站
func (c *Client) StartSite(id int64, name string) (RespMSG, error) {
	data := map[string][]string{
		"id":   {strconv.FormatInt(id, 10)},
		"name": {name},
	}
	resp, _ := c.btAPI(data, "/site?action=SiteStart")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// SetSiteEdate 设置网站过期时间 格式 “0000-00-00”（全 0 为永久）
func (c *Client) SetSiteEdate(id int64, edate string) (RespMSG, error) {
	data := map[string][]string{
		"id":    {strconv.FormatInt(id, 10)},
		"edate": {edate},
	}
	resp, _ := c.btAPI(data, "/site?action=SetEdate")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// SetSitePS 设置网站备注
func (c *Client) SetSitePS(id int64, ps string) (RespMSG, error) {
	data := map[string][]string{
		"id": {strconv.FormatInt(id, 10)},
		"ps": {ps},
	}
	resp, _ := c.btAPI(data, "/data?action=setPs&table=sites")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// GetSiteBackups 获取网站备份列表
func (c *Client) GetSiteBackups(params *ReqSiteBackups) (RespSiteBackups, error) {
	data := map[string][]string{
		"p":      {strconv.FormatInt(params.P, 10)},
		"limit":  {strconv.FormatInt(params.Limit, 10)},
		"type":   {strconv.FormatInt(params.Type, 10)},
		"tojs":   {params.ToJS},
		"search": {strconv.FormatInt(params.Search, 10)},
	}
	resp, err := c.btAPI(data, "/data?action=getData&table=backup")
	// fmt.Println(string(resp))
	if err != nil {
		return RespSiteBackups{}, err
	}
	var dec RespSiteBackups
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespSiteBackups{}, err
	}
	return dec, nil
}

// SiteBackup 创建网站备份
func (c *Client) SiteBackup(id int64) (RespMSG, error) {
	data := map[string][]string{
		"id": {strconv.FormatInt(id, 10)},
	}
	resp, _ := c.btAPI(data, "/site?action=ToBackup")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// DeleteSiteBackup 删除网站备份
func (c *Client) DeleteSiteBackup(id int64) (RespMSG, error) {
	data := map[string][]string{
		"id": {strconv.FormatInt(id, 10)},
	}
	resp, _ := c.btAPI(data, "/site?action=DelBackup")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// GetSiteDomains 获取网站域名列表
func (c *Client) GetSiteDomains(keyWords ...string) (SiteDomains, error) {
	data := map[string][]string{
		"list": {"true"},
	}
	if len(keyWords) != 0 {
		data["search"] = keyWords
	}
	resp, err := c.btAPI(data, "/data?action=getData&table=domain")
	if err != nil {
		return SiteDomains{}, err
	}
	var dec SiteDomains
	if err := json.Unmarshal(resp, &dec); err != nil {
		return SiteDomains{}, err
	}
	return dec, nil
}

// AddDomain 网站添加域名
// id 网站ID-必填
// webname 网站名称-必填
// domain 域名-必填
func (c *Client) AddDomain(id int64, webname string, domain string) (RespMSG, error) {
	data := map[string][]string{
		"id":      {strconv.FormatInt(id, 10)},
		"webname": {webname},
		"domain":  {domain},
	}
	resp, _ := c.btAPI(data, "/site?action=AddDomain")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// DelDomain 网站删除域名
// id 网站ID-必填
// webname 网站名称-必填
// domain 域名-必填
func (c *Client) DelDomain(id int64, webname string, domain string, port int64) (RespMSG, error) {
	data := map[string][]string{
		"id":      {strconv.FormatInt(id, 10)},
		"webname": {webname},
		"domain":  {domain},
		"port":    {strconv.FormatInt(port, 10)},
	}
	resp, _ := c.btAPI(data, "/site?action=DelDomain")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// GetRewriteList 获取网站可选伪静态列表
func (c *Client) GetRewriteList(siteName string) (RewriteList, error) {
	data := map[string][]string{
		"siteName": {siteName},
	}
	resp, err := c.btAPI(data, "/site?action=GetRewriteList")
	if err != nil {
		return RewriteList{}, err
	}
	var dec RewriteList
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RewriteList{}, err
	}
	return dec, nil
}

// GetFile 获取文件
func (c *Client) GetFile(path string) (RespGetFile, error) {
	data := map[string][]string{
		"path": {path},
	}
	resp, err := c.btAPI(data, "/files?action=GetFileBody")
	if err != nil {
		return RespGetFile{}, err
	}
	var dec RespGetFile
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespGetFile{}, err
	}
	return dec, nil
}

// SetFile 修改文件（无法新建文件）
func (c *Client) SetFile(path string, body string) (RespMSG, error) {
	data := map[string][]string{
		"path":     {path},
		"data":     {body},
		"encoding": {"utf-8"},
	}
	resp, _ := c.btAPI(data, "/files?action=SaveFileBody")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// GetDirUserINI 取回防跨站配置/运行目录/日志开关状态/可设置的运行目录列表/密码访问状态
func (c *Client) GetDirUserINI(id int64, path string) (RespUserINI, error) {
	data := map[string][]string{
		"id":   {strconv.FormatInt(id, 10)},
		"path": {path},
	}
	resp, err := c.btAPI(data, "/site?action=GetDirUserINI")
	if err != nil {
		return RespUserINI{}, err
	}
	var dec RespUserINI
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespUserINI{}, err
	}
	return dec, nil
}

// SetDirUserINI 设置防跨站状态（自动取反）
func (c *Client) SetDirUserINI(path string) (RespMSG, error) {
	data := map[string][]string{
		"path": {path},
	}
	resp, _ := c.btAPI(data, "/site?action=SetDirUserINI")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// SetLogsOpen 设置是否写访问日志
func (c *Client) SetLogsOpen(id int64) (RespMSG, error) {
	data := map[string][]string{
		"id": {strconv.FormatInt(id, 10)},
	}
	resp, _ := c.btAPI(data, "/site?action=logsOpen")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// SetPath 修改网站根目录
func (c *Client) SetPath(id int64, path string) (RespMSG, error) {
	data := map[string][]string{
		"id":   {strconv.FormatInt(id, 10)},
		"path": {path},
	}
	resp, _ := c.btAPI(data, "/site?action=SetPath")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// SetRunPath 修改网站运行目录 path 填相对目录 比如 "/public"
func (c *Client) SetRunPath(id int64, path string) (RespMSG, error) {
	data := map[string][]string{
		"id":      {strconv.FormatInt(id, 10)},
		"runPath": {path},
	}
	resp, _ := c.btAPI(data, "/site?action=SetSiteRunPath")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// SetHasPwd 打开并设置网站密码访问
func (c *Client) SetHasPwd(id int64, user string, pwd string) (RespMSG, error) {
	data := map[string][]string{
		"id":       {strconv.FormatInt(id, 10)},
		"username": {user},
		"password": {pwd},
	}
	resp, _ := c.btAPI(data, "/site?action=SetHasPwd")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// CloseHasPwd 关闭网站密码访问
func (c *Client) CloseHasPwd(id int64) (RespMSG, error) {
	data := map[string][]string{
		"id": {strconv.FormatInt(id, 10)},
	}
	resp, _ := c.btAPI(data, "/site?action=CloseHasPwd")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// GetLimitNet 获取流量限制相关配置（仅支持 nginx）
func (c *Client) GetLimitNet(id int64) (RespLimitNet, error) {
	data := map[string][]string{
		"id": {strconv.FormatInt(id, 10)},
	}
	resp, err := c.btAPI(data, "/site?action=GetLimitNet")
	if err != nil {
		return RespLimitNet{}, errors.New(string(resp))
	}
	var dec RespLimitNet
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespLimitNet{}, err
	}
	return dec, nil
}

// SetLimitNet 开启或保存流量限制配置（仅支持 nginx）
func (c *Client) SetLimitNet(id int64, perServer int64, perIP int64, limitRate int64) (RespMSG, error) {
	data := map[string][]string{
		"id":         {strconv.FormatInt(id, 10)},
		"perserver":  {strconv.FormatInt(perServer, 10)},
		"perip":      {strconv.FormatInt(perIP, 10)},
		"limit_rate": {strconv.FormatInt(limitRate, 10)},
	}
	resp, _ := c.btAPI(data, "/site?action=SetLimitNet")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// CloseLimitNet 关闭流量限制
func (c *Client) CloseLimitNet(id int64) (RespMSG, error) {
	data := map[string][]string{
		"id": {strconv.FormatInt(id, 10)},
	}
	resp, _ := c.btAPI(data, "/site?action=CloseLimitNet")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// GetIndex 取默认文档信息
func (c *Client) GetIndex(id int64) (string, error) {
	data := map[string][]string{
		"id": {strconv.FormatInt(id, 10)},
	}
	resp, err := c.btAPI(data, "/site?action=GetIndex")
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

// SetIndex 设置默认文档 ep. Index : "index.php,index.html,index.htm,default.php,default.htm,default.html"
func (c *Client) SetIndex(id int64, Index string) (RespMSG, error) {
	data := map[string][]string{
		"id":    {strconv.FormatInt(id, 10)},
		"Index": {Index},
	}
	resp, _ := c.btAPI(data, "/site?action=SetIndex")
	var dec RespMSG
	if err := json.Unmarshal(resp, &dec); err != nil {
		return RespMSG{}, err
	}
	return dec, nil
}

// MD5 Generate 32-bit MD5 strings
func MD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
