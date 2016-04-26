package neutrino

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/neutrinoapp/neutrino/src/common/log"
	"github.com/neutrinoapp/neutrino/src/common/models"
)

type ApiClient struct {
	BaseUrl, Token, ClientId, AppId, Origin string
	NotifyRealTime                          bool
	Filter                                  map[string]interface{}
}

var clientCache map[string]*ApiClient

func init() {
	clientCache = make(map[string]*ApiClient)
}

var (
	_httpAddr, _wsAddr, _token, _origin string
)

func InitClient(httpAddr, wsAddr, token, origin string) {
	_httpAddr = httpAddr
	_wsAddr = wsAddr
	_token = token
	_origin = origin
}

func NewApiClientClean() *ApiClient {
	return NewApiClientCached("")
}

func NewApiClientCached(appId string) *ApiClient {
	if clientCache[appId] == nil {
		clientCache[appId] = NewApiClient(appId)
	}

	return clientCache[appId]
}

func NewApiClient(appId string) *ApiClient {
	url := _httpAddr

	return &ApiClient{
		BaseUrl:        url,
		Token:          _token,
		ClientId:       "",
		NotifyRealTime: false,
		AppId:          appId,
		Origin:         _origin,
	}
}

func (c *ApiClient) SendRequest(url, method string, body interface{}, isArray bool) (interface{}, error) {
	log.Info(
		"Sending request",
		"BaseUrl:", c.BaseUrl,
		"Url:", url,
		"Method:", method,
		"Body:", body,
		"Token:", c.Token,
		"AppId:", c.AppId,
		"NotifyRealtime", c.NotifyRealTime,
	)

	var bodyStr = ""
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		bodyStr = string(b)
	}

	req, err := http.NewRequest(method, c.BaseUrl+url, strings.NewReader(bodyStr))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	opts := models.Options{}

	//todo: in app and not in app token
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	if c.ClientId != "" {
		opts.ClientId = &c.ClientId
	}

	opts.Notify = &c.NotifyRealTime
	opts.Filter = c.Filter

	optsS, err := opts.String()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	req.Header.Set("NeutrinoOptions", optsS)

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	if res == nil {
		log.Error("Unknown error")
		return nil, nil
	}

	if res.StatusCode != http.StatusOK {
		log.Info(res, err)
		return nil, err
	}

	defer res.Body.Close()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	bodyRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if string(bodyRes) == "" {
		log.Info("Empty body response!")
		return nil, nil
	}

	var result interface{}
	log.Info("API response: ", string(bodyRes))
	if isArray {
		jsonArray := make([]map[string]interface{}, 0)
		err = json.Unmarshal(bodyRes, &jsonArray)
		result = jsonArray
	} else {
		m := map[string]interface{}{}
		err = json.Unmarshal(bodyRes, &m)
		result = m
	}

	if err != nil {
		log.Error(err)
		return nil, err
	}

	return result, nil
}

func (c *ApiClient) CreateApp(name string) (string, error) {
	res, err := c.SendRequest("app", "POST", map[string]interface{}{
		"name": name,
	}, false)

	if res == nil {
		return "", err
	}

	return res.(map[string]interface{})["id"].(string), nil
}

func (c *ApiClient) GetApps() ([]map[string]interface{}, error) {
	res, err := c.SendRequest("app", "GET", nil, true)
	if res == nil {
		return nil, err
	}

	return res.([]map[string]interface{}), nil
}

func (c *ApiClient) AppRegister(email, password string) error {
	_, err := c.SendRequest("app/"+c.AppId+"/register", "POST", map[string]interface{}{
		"email":    email,
		"password": password,
	}, false)

	return err
}

func (c *ApiClient) Register(email, password string) error {
	_, err := c.SendRequest("register", "POST", map[string]interface{}{
		"email":    email,
		"password": password,
	}, false)

	return err
}

func (c *ApiClient) AppLogin(email, password string) (string, error) {
	res, err := c.SendRequest("app/"+c.AppId+"/login", "POST", map[string]interface{}{
		"email":    email,
		"password": password,
	}, false)

	if res == nil {
		return "", err
	}

	c.Token = res.(map[string]interface{})["token"].(string)

	return c.Token, nil
}

func (c *ApiClient) Login(email, password string) (string, error) {
	res, err := c.SendRequest("login", "POST", map[string]interface{}{
		"email":    email,
		"password": password,
	}, false)

	if res == nil {
		return "", err
	}

	c.Token = res.(map[string]interface{})["token"].(string)

	return c.Token, nil
}

func (c *ApiClient) CreateItem(t string, m map[string]interface{}) (map[string]interface{}, error) {
	res, err := c.SendRequest("app/"+c.AppId+"/data/"+t, "POST", m, false)
	if res == nil {
		return nil, err
	}

	return res.(map[string]interface{}), err
}

func (c *ApiClient) UpdateItem(t, id string, m map[string]interface{}) (map[string]interface{}, error) {
	res, err := c.SendRequest("app/"+c.AppId+"/data/"+t+"/"+id, "PUT", m, false)
	if res == nil {
		return nil, err
	}

	return res.(map[string]interface{}), err
}

func (c *ApiClient) DeleteItem(t, id string) (map[string]interface{}, error) {
	res, err := c.SendRequest("app/"+c.AppId+"/data/"+t+"/"+id, "DELETE", nil, false)
	if res == nil {
		return nil, err
	}

	return res.(map[string]interface{}), err
}

func (c *ApiClient) GetCollections() ([]string, error) {
	res, err := c.SendRequest("app/"+c.AppId+"/data", "GET", nil, false)
	if res == nil {
		return nil, err
	}

	return res.([]string), err
}

func (c *ApiClient) GetItem(t, id string) (map[string]interface{}, error) {
	res, err := c.SendRequest("app/"+c.AppId+"/data/"+t+"/"+id, "GET", nil, false)
	if res == nil {
		return nil, err
	}

	return res.(map[string]interface{}), err
}

func (c *ApiClient) GetItems(t string) ([]map[string]interface{}, error) {
	res, err := c.SendRequest("app/"+c.AppId+"/data/"+t, "GET", nil, true)
	if res == nil {
		return nil, err
	}

	return res.([]map[string]interface{}), err
}
