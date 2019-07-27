# swagger.yaml

使用swagger编辑器

	docker run -d -p 8081:8080 swaggerapi/swagger-editor

swagger文档，https://swagger.io/docs/specification/2-0/basic-structure/

编辑测试：

1. 首先启动swagger-ui

    sudo docker pull swaggerapi/swagger-ui
    sudo docker run -d -p 80:8080 swaggerapi/swagger-ui

2. 用python启动本地的http server，这是允许CORS的SimpleHTTPServer，监听在8000端口

   python server.py

3. 用浏览器打开swagger-ui的URL：http://<swagger_ui_host>?url=http://<simple_http_server_host>:8000/index.yaml
