package server

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"bitbucket.org/linkernetworks/vortex/src/entity"
	response "bitbucket.org/linkernetworks/vortex/src/net/http"
	"bitbucket.org/linkernetworks/vortex/src/net/http/query"
	"bitbucket.org/linkernetworks/vortex/src/web"
	"github.com/linkernetworks/logger"
	"github.com/linkernetworks/utils/timeutils"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func CreateNetworkHandler(ctx *web.Context) {
	as, req, resp := ctx.ServiceProvider, ctx.Request, ctx.Response

	network := entity.Network{}

	if err := req.ReadEntity(&network); err != nil {
		logger.Error(err)
		response.BadRequest(req.Request, resp.ResponseWriter, err)
		return
	}

	session := as.Mongo.NewSession()
	defer session.Close()

	// Check whether this displayname has been used
	query := bson.M{"displayName": network.DisplayName}
	existed := entity.Network{}
	if err := session.FindOne(entity.NetworkCollectionName, query, &existed); err != nil {
		if err.Error() != mgo.ErrNotFound.Error() {
			logger.Error(err)
			response.InternalServerError(req.Request, resp.ResponseWriter, err)
			return
		}
	}
	if len(existed.ID) > 1 {
		response.Conflict(req.Request, resp, fmt.Errorf("displayName: %s already existed", network.DisplayName))
		return
	}

	// Check whether this Interface has been used
	query = bson.M{"node": network.Node, "interface": network.Interface}
	existed = entity.Network{}
	if err := session.FindOne(entity.NetworkCollectionName, query, &existed); err != nil {
		if err.Error() != mgo.ErrNotFound.Error() {
			logger.Error(err)
			response.InternalServerError(req.Request, resp.ResponseWriter, err)
			return
		}
	}
	if len(existed.ID) > 1 {
		response.Conflict(req.Request, resp, fmt.Errorf("interface %s on the Node %s has already be used", network.Interface, network.Node))
		return
	}

	// Check whether this bridge has been used
	query = bson.M{"bridgeName": network.BridgeName}
	existed = entity.Network{}
	if err := session.FindOne(entity.NetworkCollectionName, query, &existed); err != nil {
		if err.Error() != mgo.ErrNotFound.Error() {
			logger.Error(err)
			response.InternalServerError(req.Request, resp.ResponseWriter, err)
			return
		}
	}
	if len(existed.ID) > 1 {
		response.Conflict(req.Request, resp, fmt.Errorf("bridgeName: %s already existed", network.BridgeName))
		return
	}

	network.ID = bson.NewObjectId()
	network.CreatedAt = timeutils.Now()

	if err := session.Insert(entity.NetworkCollectionName, &network); err != nil {
		logger.Error(err)
		response.InternalServerError(req.Request, resp.ResponseWriter, err)
		return
	}

	resp.WriteEntity(ActionResponse{
		Error:   false,
		Message: "Create success",
	})
}

func ListNetworkHandler(ctx *web.Context) {
	as, req, resp := ctx.ServiceProvider, ctx.Request, ctx.Response

	var pageSize = 10
	query := query.New(req.Request.URL.Query())

	page, err := query.Int("page", 1)
	if err != nil {
		logger.Error(err)
		response.BadRequest(req.Request, resp.ResponseWriter, err)
		return
	}
	pageSize, err = query.Int("page_size", pageSize)
	if err != nil {
		response.BadRequest(req.Request, resp.ResponseWriter, err)
		return
	}

	session := as.Mongo.NewSession()
	defer session.Close()

	networks := []entity.Network{}

	var c = session.C(entity.NetworkCollectionName)
	var q *mgo.Query

	selector := bson.M{}
	q = c.Find(selector).Sort("_id").Skip((page - 1) * pageSize).Limit(pageSize)

	err = q.All(&networks)
	if err != nil {
		logger.Error(err)
		if err == mgo.ErrNotFound {
			response.NotFound(req.Request, resp.ResponseWriter, err)
			return
		}
		response.InternalServerError(req.Request, resp.ResponseWriter, err)
		return
	}

	count, err := session.Count(entity.NetworkCollectionName, bson.M{})
	if err != nil {
		logger.Error(err)
	}
	totalPages := int(math.Ceil(float64(count) / float64(pageSize)))
	resp.AddHeader("X-Total-Count", strconv.Itoa(count))
	resp.AddHeader("X-Total-Pages", strconv.Itoa(totalPages))
	resp.WriteEntity(networks)
}

func GetNetworkHandler(ctx *web.Context) {
	as, req, resp := ctx.ServiceProvider, ctx.Request, ctx.Response

	id := req.PathParameter("id")

	session := as.Mongo.NewSession()
	defer session.Close()
	c := session.C(entity.NetworkCollectionName)

	var network entity.Network
	err := c.FindId(bson.ObjectIdHex(id)).One(&network)
	if err != nil {
		logger.Error(err)
		if err == mgo.ErrNotFound {
			response.NotFound(req.Request, resp.ResponseWriter, err)
			return
		}
		response.InternalServerError(req.Request, resp.ResponseWriter, err)
		return
	} else {
		resp.WriteEntity(network)
	}
}

func DeleteNetworkHandler(ctx *web.Context) {
	as, req, resp := ctx.ServiceProvider, ctx.Request, ctx.Response

	id := req.PathParameter("id")

	session := as.Mongo.NewSession()
	defer session.Close()

	network := entity.Network{}
	q := bson.M{"_id": bson.ObjectIdHex(id)}

	if err := session.FindOne(entity.NetworkCollectionName, q, &network); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			logger.Error(err)
			response.NotFound(req.Request, resp.ResponseWriter, fmt.Errorf("the network: %v doesn't exist", id))
			return
		}
		response.InternalServerError(req.Request, resp.ResponseWriter, err)
		return
	}

	if err := session.Remove(entity.NetworkCollectionName, "_id", network.ID); err != nil {
		logger.Error(err)
		response.InternalServerError(req.Request, resp.ResponseWriter, err)
		return
	}

	resp.WriteEntity(ActionResponse{
		Error:   false,
		Message: "Delete success",
	})
}

func UpdateNetworkHandler(ctx *web.Context) {
	as, req, resp := ctx.ServiceProvider, ctx.Request, ctx.Response

	id := req.PathParameter("id")

	session := as.Mongo.NewSession()
	defer session.Close()

	network := entity.Network{}
	q := bson.M{"_id": bson.ObjectIdHex(id)}

	if err := session.FindOne(entity.NetworkCollectionName, q, &network); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			logger.Error(err)
			response.NotFound(req.Request, resp.ResponseWriter, fmt.Errorf("the network: %v doesn't exist", id))
			return
		}
		response.InternalServerError(req.Request, resp.ResponseWriter, err)
		return
	}

	updatedNetwork := entity.Network{}
	err := req.ReadEntity(&updatedNetwork)
	if err != nil {
		logger.Error(err)
		response.BadRequest(req.Request, resp.ResponseWriter, err)
		return
	}

	checkNetwork := entity.Network{}
	checkNetwork.DisplayName = updatedNetwork.DisplayName
	if !reflect.DeepEqual(updatedNetwork, checkNetwork) {
		response.BadRequest(req.Request, resp.ResponseWriter, fmt.Errorf("only DisplayName can be changed"))
		return
	}

	err = session.UpdateById(entity.NetworkCollectionName, network.ID, updatedNetwork)
	if err != nil {
		logger.Error(err)
		if err == mgo.ErrNotFound {
			response.NotFound(req.Request, resp.ResponseWriter, err)
			return
		}
		response.InternalServerError(req.Request, resp.ResponseWriter, err)
		return
	}
	resp.WriteEntity(updatedNetwork)
}
