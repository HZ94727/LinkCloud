package main

import (
	"log"

	"gitea.com/hz/linkcloud/config"
	"gitea.com/hz/linkcloud/database"
	"gitea.com/hz/linkcloud/routes"
)

func main() {
	// 初始化配置
	config.Init()

	// 初始化数据库连接池
	if err := database.Init(); err != nil {
		log.Fatal("数据库初始化失败: ", err)
	}
	defer database.Close() // 程序退出时关闭连接

	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)

	// 启动服务
	// gin.SetMode(gin.ReleaseMode)
	r := routes.SetupRouter()
	r.Run(":8080")
}
