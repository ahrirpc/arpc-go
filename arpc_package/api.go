//119299-110-11 22:00:00

package api

import (
	"arpc-go/net"
	"arpc-go/server"
	"encoding/json"
)

type ApiRequestV1 struct {
    UserId int
}

func (b *ApiRequestV1) New(user_id int) {
	b.UserId = user_id
}

func (b *ApiRequestV1) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

func (b *ApiRequestV1) Deserialize(data []byte) error {
	return json.Unmarshal(data, b)
}

type ApiResponseV1 struct {
    UserId   int
    Username string
}

func (b *ApiResponseV1) New(user_id int, username string) {
	b.UserId = user_id
    b.Username = username
}

func (b *ApiResponseV1) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

func (b *ApiResponseV1) Deserialize(data []byte) error {
	return json.Unmarshal(data, b)
}

type client struct {
	conn net.ArpcConn
}

type Client interface {
    GetUserV1(*ApiRequestV1) (*ApiResponseV1, error)
}

func (c *client) GetUserV1(request *ApiRequestV1) (*ApiResponseV1, error) {
	req_bytes, err := request.Serialize()
	if err != nil {
		return nil, err
	}
	data, err := net.Handle("GetUserV1", req_bytes, c.conn)
	if err != nil {
		return nil, err
	}
	response := &ApiResponseV1{}
	err = response.Deserialize(data)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func RegisterGetUserV1(s *server.Server, i Client) {
	s.Register("GetUserV1", func(request []byte, _ net.ArpcConn) ([]byte, error) {
		req := &ApiRequestV1{}
		err := req.Deserialize(request)
		if err != nil {
			return nil, err
		}
		response, err := i.GetUserV1(req)
		if err != nil {
			return nil, err
		}
		return response.Serialize()
	})
}

func NewClient(c net.ArpcConn) Client {
	return &client{c}
}
