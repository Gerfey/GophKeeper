package client

type TestClient interface {
	GetConfigDir() (string, error)
}

var _ TestClient = &Client{}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) GetToken() string {
	return c.token
}

func (c *Client) SetSalt(salt []byte) {
	c.salt = salt
}

func (c *Client) SetUsername(username string) {
	c.username = username
}

func (c *Client) GetUsername() string {
	return c.username
}

func (c *Client) SetConfigPath(path string) {
	c.configPath = path
}

func (c *Client) SetHTTPClient(httpClient HTTPClient) {
	c.httpClient = httpClient
}
