package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

func (a *Workloads) create(request *http.Request) (interface{}, mw.Response) {
	var deployment gridtypes.Deployment
	if err := json.NewDecoder(request.Body).Decode(&deployment); err != nil {
		return nil, mw.BadRequest(err)
	}

	if err := deployment.Valid(); err != nil {
		return nil, mw.BadRequest(err)
	}

	//TODO: signature validation

	// userID := mw.UserID(request.Context())
	// if workload.User != userID {
	// 	return nil, mw.UnAuthorized(fmt.Errorf("invalid user id in request body doesn't match http signature"))
	// }
	// userPK := mw.UserPublicKey(request.Context())

	// if err := workload.Verify(userPK); err != nil {
	// 	return nil, mw.UnAuthorized(err)
	// }
	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Minute)
	defer cancel()

	err := a.engine.Provision(ctx, deployment)
	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

func (a *Workloads) parseIDs(request *http.Request) (twin, id uint32, err error) {
	twinVar := mux.Vars(request)["twin"]
	if len(twinVar) == 0 {
		return twin, id, fmt.Errorf("invalid twin id format")
	}
	idVar := mux.Vars(request)["id"]
	if len(idVar) == 0 {
		return twin, id, fmt.Errorf("invalid id format")
	}

	twinU, err := strconv.ParseUint(twinVar, 10, 32)
	if err != nil {
		return twin, id, err
	}

	idU, err := strconv.ParseUint(idVar, 10, 32)
	if err != nil {
		return twin, id, err
	}

	return uint32(twinU), uint32(idU), nil
}

func (a *Workloads) delete(request *http.Request) (interface{}, mw.Response) {

	twin, id, err := a.parseIDs(request)
	if err != nil {
		return nil, mw.BadRequest(err)
	}
	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Minute)
	defer cancel()

	err = a.engine.Deprovision(ctx, twin, id, "requested by user")
	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

func (a *Workloads) get(request *http.Request) (interface{}, mw.Response) {

	twin, id, err := a.parseIDs(request)
	if err != nil {
		return nil, mw.BadRequest(err)
	}

	deployment, err := a.engine.Storage().Get(twin, id)
	if err == provision.ErrDeploymentNotExists {
		return nil, mw.NotFound(fmt.Errorf("workload not found"))
	} else if err != nil {
		return nil, mw.Error(err)
	}

	// if deployment.User != mw.UserID(request.Context()) {
	// 	return nil, mw.UnAuthorized(fmt.Errorf("access denied"))
	// }

	return deployment, nil
}
