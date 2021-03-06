/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/utils"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
)

func genKeyPair(c *gin.Context) {
	r := struct {
		Password string `json:"password" form:"password"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// save key to persistence
	developer := getDeveloperID(c)

	p, err := model.AddNewPrivateKey(model.GetDB(c), developer)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrGenerateKeyPairFailed)
		return
	}

	// set as main account
	err = model.SetIfNoMainAccount(model.GetDB(c), developer, p.Account)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrSetMainAccountFailed)
		return
	}

	keyBytes, err := kms.EncodePrivateKey(p.Key, []byte(r.Password))
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrEncodePrivateKeyFailed)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"account": p.Account,
		"key":     string(keyBytes),
	})
}

func uploadKeyPair(c *gin.Context) {
	r := struct {
		Key      string `json:"key" form:"key" binding:"required"`
		Password string `json:"password" form:"password"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// decode key
	key, err := kms.DecodePrivateKey([]byte(r.Key), []byte(r.Password))
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrInvalidPrivateKeyUploaded)
		return
	}

	// save key to persistence
	developer := getDeveloperID(c)

	p, err := model.SavePrivateKey(model.GetDB(c), developer, key)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrSavePrivateKeyFailed)
		return
	}

	// set as main account
	err = model.SetIfNoMainAccount(model.GetDB(c), developer, p.Account)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrSetMainAccountFailed)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"account": p.Account,
	})
}

func deleteKeyPair(c *gin.Context) {
	r := struct {
		Account utils.AccountAddress `json:"account" form:"account" uri:"account" binding:"required,len=64"`
		Force   bool                 `json:"force" form:"force"`
	}{}

	// ignore validation, check in later ShouldBind
	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// check and delete private key
	developer := getDeveloperID(c)
	db := model.GetDB(c)

	account, err := model.GetAccount(db, developer, r.Account)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrGetAccountFailed)
		return
	}

	// check account for projects
	var projects []*model.Project
	projects, err = model.GetUserProjects(db, developer, account.ID)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrGetProjectsFailed)
		return
	}

	if len(projects) > 0 {
		if r.Force {
			err = model.DeleteProjects(db, projects...)
			if err != nil {
				_ = c.Error(err)
				abortWithError(c, http.StatusInternalServerError, ErrDeleteProjectsFailed)
				return
			}
		} else {
			err = ErrKeyPairHasRelatedProjects
			abortWithError(c, http.StatusBadRequest, err)
			return
		}
	}

	p, err := model.DeletePrivateKey(db, developer, r.Account)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrDeletePrivateKeyFailed)
		return
	}

	err = model.FixDeletedMainAccount(db, developer, p.ID)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrUnbindMainAccountFailed)
		return
	}

	responseWithData(c, http.StatusOK, nil)
}

func downloadKeyPair(c *gin.Context) {
	r := struct {
		Account  utils.AccountAddress `json:"account" form:"account" uri:"account" binding:"required,len=64"`
		Password string               `json:"password" form:"password"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// check private key
	developer := getDeveloperID(c)

	p, err := model.GetPrivateKey(model.GetDB(c), developer, r.Account)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrGetAccountFailed)
		return
	}

	privateKeyBytes, err := kms.EncodePrivateKey(p.Key, []byte(r.Password))
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrEncodePrivateKeyFailed)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"key": string(privateKeyBytes),
	})
}
