<div align="center">
  <img height="150px" src="./assets/logo.png"></img>
</div>

<h1 align="center">go-emby2openlist</h1>

<div align="center">
  <a href="https://github.com/siseboy/go-emby2openlist/releases/latest"><img src="https://img.shields.io/github/v/tag/siseboy/go-emby2openlist"></img></a>
  <a href="https://hub.docker.com/r/siseboy/go-emby2openlist/tags"><img src="https://img.shields.io/docker/image-size/siseboy/go-emby2openlist/latest"></img></a>
  <a href="https://hub.docker.com/r/siseboy/go-emby2openlist/tags"><img src="https://img.shields.io/docker/pulls/siseboy/go-emby2openlist"></img></a>
  <a href="https://github.com/siseboy/go-emby2openlist/releases/latest"><img src="https://img.shields.io/github/downloads/siseboy/go-emby2openlist/total"></img></a>
  <img src="https://img.shields.io/github/stars/siseboy/go-emby2openlist"></img>
  <img src="https://img.shields.io/github/license/siseboy/go-emby2openlist"></img>
</div>

<div align="center">
  Go 语言编写的 Emby + Navidrome + OpenList 网盘直链反向代理服务。
</div>

## 📢 重要更新 v2.4.0

**🎉 项目已改为适配个人使用习惯，删除大量功能！**

如果你有更复杂的需求, 推荐使用原项目或功能更完善的反向代理服务：[bpking1/embyExternalUrl](https://github.com/bpking1/embyExternalUrl)

## 小白必看

**网盘直链反向代理**:

正常情况下，Emby 通过磁盘挂载的形式间接读取网盘资源，走的是服务器代理模式，看一个视频时数据链路是：

> 客户端 => Emby 源服务器 => 磁盘挂载服务 => OpenList => 网盘
>
> 客户端 <= Emby 源服务器 <= 磁盘挂载服务（将视频数据加载到本地，再给 Emby 读取） <= OpenList <= 网盘

这种情况有以下局限：

1. 视频经过服务器中转，你看视频的最大加载速度就是服务器的上传带宽
2. 如果服务器性能不行，能流畅播放 1080p 就谢天谢地了，更别说 4K
3. ...

使用网盘直链反向代理后，数据链路：

> 客户端 => Emby 反代服务器 => Emby 源服务器 （请求 Emby Api 接口）
>
> 客户端 <= Emby 反代服务器 <= Emby 源服务器 （返回数据）

对于普通的 Api 接口，反代服务器将请求反代到源服务器，再将合适的结果进行缓存，返回给客户端

对于客户端来说，这一步和直连源服务器看不出差别

> 客户端 => Emby 反代服务器 => OpenList => 网盘 （请求视频直链）
>
> 客户端 <= Emby 反代服务器 <= OpenList <= 网盘 （返回视频直链，并给出重定向响应）
>
> 客户端 => 网盘（客户端拿着网盘的直链直接观看，此时已经没有服务器的事情了，故不会再消耗服务器流量）

这种方式的好处：

1. 观看时加载速度拉满（前提是有网盘会员）
2. 在客户端处解码，能不能看 4K 取决于你电视盒子的性能

## 功能

- OpenList 网盘原画直链播放

- Strm 直链播放

- [OpenList 本地目录树生成](#使用说明-openlist-本地目录树生成)
  
- websocket 代理

- 客户端防转码（转容器）

- 缓存中间件，实际使用体验不会比直连源服务器差

- 字幕缓存（字幕缓存时间固定 30 天）

- 直链缓存

- 大接口缓存（OpenList 转码资源是通过代理并修改 PlaybackInfo 接口实现，请求比较耗时，每次大约 2~3 秒左右，目前已经利用 Go 语言的并发优势，尽力地将接口处理逻辑异步化，快的话 1 秒即可请求完成，该接口的缓存时间目前固定为 12 小时，后续如果出现异常再作调整）


## 使用 Docker 部署安装

### 使用现有镜像

1. 准备配置

> 示例配置为完整版配置，首次部署可以参照下方优先跑通程序，再按需补充其他配置，保存并命名为 `config.yml`
```yaml
emby:
  host: http://172.17.0.1:8096            # emby 访问地址
  mount-path: /data                          # rclone/cd2 挂载的本地磁盘路径, 如果 emby 是容器部署, 这里要配的就是容器内部的挂载路径

# navidrome访问配置
navidrome:
  # Navidrome服务访问地址
  host: http://172.17.0.1:4533

# 该配置仅针对通过磁盘挂载方式接入的网盘, 如果你使用的是 strm, 可忽略该配置
openlist:
  host: http://172.17.0.1:5244            # openlist 访问地址
  token: openlist-xxxxx                         # openlist api key 可以在 openlist 管理后台查看

path:
  # emby 挂载路径和 openlist 真实路径之间的前缀映射
  # 冒号左边表示本地挂载路径, 冒号右边表示 alist 的真实路径
  # 这个配置请再三确认配置正确, 可以减少很多不必要的网络请求
  emby2openlist: 
    - /movie:/电影
    - /music:/音乐
    - /show:/综艺
    - /series:/电视剧
    - /sport:/运动
    - /animation:/动漫
```

2. 创建 docker-compose 文件

在配置相同目录下，创建 `docker-compose.yml` 粘贴以下代码：

```yaml
version: "3.1"
services:
  go-emby2openlist:
    image: siseboy/go-emby2openlist:latest
    environment:
      - TZ=Asia/Shanghai
      - GIN_MODE=release
    container_name: go-emby2openlist
    restart: always
    volumes:
      - ./config.yml:/app/config.yml
      - ./localtree:/app/localtree
    ports:
      - 8090:8090 # navidrome
      - 8095:8095 # emby
```

3. 运行容器

```shell
docker-compose up -d --build
```

## 使用说明 OpenList 本地目录树生成

监控扫描 OpenList 目录树变更，在本地磁盘中生成并维护相应结构的目录树，可供 Emby 服务器直接扫描入库，并配合本项目进行直链反代，支持传统 Strm 文件以及附带元数据的虚拟文件生成。

> ⚠️ **提示：**
>
> 程序利用 Go 的并发优势，加快了扫描 OpenList 的速度！

### 使用步骤

1. 修改配置，按照自己的需求配置好 `openlist.local-tree-gen` 属性
2. 修改 `docker-compose.yml` 文件，将容器目录 `/app/localtree` 映射到宿主机中
3. 运行程序 开始自动扫描生成目录树
4. 将宿主机的目录树路径，映射到 Emby 容器中，即可扫描入库


### 其他配置

| 属性名                     | 描述                                                         | 示例值        |
| -------------------------- | :----------------------------------------------------------- | ------------- |
| `openlist`                 | ---                                                          | ---           |
| > `local-tree-gen`         | ---                                                          | ---           |
| >> `auto-remove-max-count` | 此配置相当于为本地目录树加了个保险措施，防止 openlist 存储挂载出现异常后，程序误以为远程文件被删除，而将本地已扫描完成的目录树清空的情况。<br /><br />具体配置值需以自己 openlist 的总文件数为参考（可留意首次全量扫描目录树后的日志输出），建议配置为总文件数的 3/4 左右大小，当程序即将要删除的文件数目超过这个数值时，会停止删除操作，并在日志中输出警告 | `6000`        |
| >> `refresh-interval`      | 本地目录树刷新间隔，单位：分钟                               | `60`          |
| >> `scan-prefixes`         | 指定 openlist 要扫描到本地的前缀列表，没有配置则默认全量扫描 | ---           |
| >> `ignore-containers`     | 忽略指定容器，避免触发源文件下载                             | `jpg,png,nfo` |

## Star History

<a href="https://star-history.com/#siseboy/go-emby2openlist&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=siseboy/go-emby2openlist&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=siseboy/go-emby2openlist&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=siseboy/go-emby2openlist&type=Date" />
 </picture>
</a>