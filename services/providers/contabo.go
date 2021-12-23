package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/pkg/errors"

	"github.com/urfave/cli"

	tt "github.com/webtor-io/node-manager/services/providers/contabo"
	t "github.com/webtor-io/node-manager/services/types"

	"github.com/google/uuid"
)

const (
	ContaboClientIdFlag     = "contabo-client-id"
	ContaboClientSecretFlag = "contabo-client-secret"
	ContaboApiUserFlag      = "contabo-api-user"
	ContaboApiPasswordFlag  = "contabo-api-password"
	ContaboAuthEndpointFlag = "contabo-auth-endpoint"
	ContaboApiEndpointFlag  = "contabo-api-endpoint"
)

func RegisterContaboFlags(f []cli.Flag) []cli.Flag {
	return append(f,
		cli.StringFlag{
			Name:   ContaboClientIdFlag,
			Usage:  "contabo client id",
			Value:  "",
			EnvVar: "CONTABO_CLIENT_ID",
		},
		cli.StringFlag{
			Name:   ContaboClientSecretFlag,
			Usage:  "contabo client secret",
			Value:  "",
			EnvVar: "CONTABO_CLIENT_SECRET",
		},
		cli.StringFlag{
			Name:   ContaboApiUserFlag,
			Usage:  "contabo api user",
			Value:  "",
			EnvVar: "CONTABO_API_USER",
		},
		cli.StringFlag{
			Name:   ContaboApiPasswordFlag,
			Usage:  "contabo api password",
			Value:  "",
			EnvVar: "CONTABO_API_PASSWORD",
		},
		cli.StringFlag{
			Name:   ContaboAuthEndpointFlag,
			Usage:  "contabo auth endpoint url",
			Value:  "https://auth.contabo.com/auth/realms/contabo/protocol/openid-connect/token",
			EnvVar: "CONTABO_AUTH_ENDPOINT",
		},
		cli.StringFlag{
			Name:   ContaboApiEndpointFlag,
			Usage:  "contabo api endpoint url",
			Value:  "https://api.contabo.com/v1/",
			EnvVar: "CONTABO_API_ENDPOINT",
		},
	)
}

var (
	NotFoundErr = errors.New("contabo instance not found")
)

type Contabo struct {
	cl            *http.Client
	tokenOnce     sync.Once
	tokenErr      error
	token         string
	instancesOnce sync.Once
	instancesErr  error
	instances     []tt.Instance
	clientId      string
	clientSecret  string
	apiUser       string
	apiPassword   string
	apiEndpoint   string
	authEndpoint  string
}

func NewContabo(c *cli.Context, cl *http.Client) *Contabo {
	return &Contabo{
		cl:           cl,
		clientId:     c.String(ContaboClientIdFlag),
		clientSecret: c.String(ContaboClientSecretFlag),
		apiUser:      c.String(ContaboApiUserFlag),
		apiPassword:  c.String(ContaboApiPasswordFlag),
		apiEndpoint:  c.String(ContaboApiEndpointFlag),
		authEndpoint: c.String(ContaboAuthEndpointFlag),
	}
}

func (s *Contabo) Reboot(ctx context.Context, n *t.Node) error {
	token, err := s.getTokenOnce(ctx)
	if err != nil {
		return err
	}
	id, err := s.getInstanceId(ctx, n, token)
	if err != nil {
		return err
	}
	return s.restart(ctx, id, token)
}

func (s *Contabo) restart(ctx context.Context, id int, token string) error {
	url := fmt.Sprintf("%vcompute/instances/%v/actions/restart", s.apiEndpoint, id)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-request-id", uuid.NewString())
	res, err := s.cl.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 201 {
		return errors.Errorf("restart action failed resp=%v status=%v", body, res.StatusCode)
	}
	return nil
}

func (s *Contabo) getInstancesOnce(ctx context.Context, token string) ([]tt.Instance, error) {
	s.instancesOnce.Do(func() {
		s.instances, s.instancesErr = s.getInstances(ctx, token)
	})
	return s.instances, s.instancesErr
}

func (s *Contabo) getInstances(ctx context.Context, token string) ([]tt.Instance, error) {
	var (
		c   = true
		res = []tt.Instance{}
		err error
		in  []tt.Instance
	)
	for i := 1; c; i++ {
		in, c, err = s.getInstancesByPage(ctx, i, token)
		if err != nil {
			return nil, err
		}
		res = append(res, in...)
	}
	return res, nil
}

func (s *Contabo) getInstancesByPage(ctx context.Context, page int, token string) ([]tt.Instance, bool, error) {
	ins := []tt.Instance{}
	url := fmt.Sprintf("%vcompute/instances?size=%v&page=%v", s.apiEndpoint, 10, page)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-request-id", uuid.NewString())
	res, err := s.cl.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, false, err
	}
	var j map[string]interface{}
	err = json.Unmarshal(body, &j)
	if err != nil {
		return nil, false, err
	}
	next := j["_links"].(map[string]interface{})["next"].(string) != ""
	for _, i := range j["data"].([]interface{}) {
		ins = append(ins, tt.Instance{
			ID: int(i.(map[string]interface{})["instanceId"].(float64)),
			IP: i.(map[string]interface{})["ipConfig"].(map[string]interface{})["v4"].(map[string]interface{})["ip"].(string),
		})
	}
	return ins, next, nil
}

func (s *Contabo) getInstanceId(ctx context.Context, n *t.Node, token string) (int, error) {
	ins, err := s.getInstancesOnce(ctx, token)
	if err != nil {
		return 0, err
	}
	for _, i := range ins {
		for _, ip := range n.IPs {
			if i.IP == ip {
				return i.ID, nil
			}
		}
	}
	return 0, NotFoundErr
}

func (s *Contabo) getToken(ctx context.Context) (string, error) {
	res, err := s.cl.PostForm(s.authEndpoint, url.Values{
		"client_id":     {s.clientId},
		"client_secret": {s.clientSecret},
		"username":      {s.apiUser},
		"password":      {s.apiPassword},
		"grant_type":    {"password"},
	})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}
	var j map[string]interface{}
	err = json.Unmarshal(body, &j)
	if err != nil {
		return "", err
	}
	at, ok := j["access_token"]
	if !ok {
		return "", errors.Errorf("failed to get accesss_token body=%v", body)
	}
	return at.(string), nil
}

func (s *Contabo) getTokenOnce(ctx context.Context) (string, error) {
	s.tokenOnce.Do(func() {
		s.token, s.tokenErr = s.getToken(ctx)
	})
	return s.token, s.tokenErr
}
