/*
   Copyright (c) 2016 VMware, Inc. All Rights Reserved.
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	"github.com/vmware/harbor/src/common/utils"
	"github.com/vmware/harbor/src/common/utils/log"

	"github.com/astaxie/beego"
	_ "github.com/astaxie/beego/session/redis"
	"github.com/robfig/cron"

	dao "github.com/vmware/harbor/src/common/daomongo"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/mongo"
	"github.com/vmware/harbor/src/ui/api"
	"github.com/vmware/harbor/src/ui/appconfig"
	_ "github.com/vmware/harbor/src/ui/auth/arrowcloud"
	_ "github.com/vmware/harbor/src/ui/auth/db"
)

const (
	adminUserID   = 1 //does not apply to mongo
	adminUserName = "admin"
)

func init() {
	c := cron.New()
	c.AddFunc("@every 10m", func() {
		if err := api.SyncRegistry(); err != nil {
			log.Error(err)
		}
	})
	c.Start()
}

func updateInitPassword(userID string, password string) error {
	queryUser := models.User{Username: userID}
	user, err := dao.GetUser(queryUser)
	if err != nil {
		return fmt.Errorf("Failed to get user, userID: %v %v", userID, err)
	}
	if user == nil {
		return fmt.Errorf("user id: %v does not exist", userID)
	}
	if user.Salt == "" {
		salt := utils.GenerateRandomString()

		user.Salt = salt
		user.Password = password
		err = dao.ChangeUserPassword(*user)
		if err != nil {
			return fmt.Errorf("Failed to update user encrypted password, userID: %v, err: %v", userID, err)
		}

		log.Infof("User id: %v updated its encypted password successfully.", userID)
	} else {
		log.Infof("User id: %v already has its encrypted password.", userID)
	}
	return nil
}

func main() {

	beego.BConfig.WebConfig.Session.SessionOn = true
	//TODO
	redisURL := os.Getenv("_REDIS_URL")
	if len(redisURL) > 0 {
		beego.BConfig.WebConfig.Session.SessionProvider = "redis"
		beego.BConfig.WebConfig.Session.SessionProviderConfig = redisURL
	}
	//
	beego.AddTemplateExt("htm")

	//dao.InitDatabase()
	mongo.InitDatabase()
	//nothing but satisfies orm module
	// orm.RegisterDataBase("default", "mysql", "root:root@tcp(mysql:3306)/registry")

	if err := updateInitPassword(adminUserName, appconfig.InitialAdminPassword()); err != nil {
		log.Error(err)
	}
	initRouters()
	if err := api.SyncRegistry(); err != nil {
		log.Error(err)
	}
	beego.Run()
}
