package socket

import (
	"context"

	socketio "github.com/googollee/go-socket.io"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// getCred 与前端约定，在 socketio 里，使用 ?session=XXX 的方式获取用户的 session
func getCred(ctx context.Context, s socketio.Conn) (string, mcclient.TokenCredential, error) {
	query, err := jsonutils.ParseQueryString(s.URL().RawQuery)
	if err != nil {
		return "", nil, errors.Wrapf(err, "ParseQueryString")
	}
	session, err := query.GetString("session")
	if err != nil {
		return "", nil, errors.Wrapf(err, "get session")
	}
	if len(session) == 0 {
		return "", nil, errors.Errorf("empty session")
	}
	// tm := clientman.NewMapTokenManagerV2() 这行注释临时保留 -rex.
	// cred := clientman.TokenMan.Get(session)
	authToken, err := clientman.Decode(session)
	if err != nil {
		return "", nil, errors.Wrap(err, "Decode")
	}
	cred, err := authToken.GetToken(ctx)
	if err != nil {
		return "", nil, errors.Wrap(err, "authToken.GetToken")
	}
	return session, cred, nil
}
