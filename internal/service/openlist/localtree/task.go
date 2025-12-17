package localtree

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/constant"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/service/openlist"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/bytess"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/https"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/logs/colors"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/trys"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/urls"
)

// FileTask 包含同步必要信息的文件结构
type FileTask struct {
	// Path 文件绝对路径, 与 openlist 对应
	Path string

	// LocalPath 文件要存入本地的路径
	LocalPath string

	// IsDir 是否是目录
	IsDir bool

	// Container 标记文件的容器
	Container string

	// Sign openlist 文件签名
	Sign string

	// Modified 文件的最后修改时间
	Modified time.Time
}

func FsGetTask(prefix string, info openlist.FsGet) FileTask {
	container := strings.TrimPrefix(strings.ToLower(filepath.Ext(info.Name)), ".")
	fp := filepath.Join(prefix, info.Name)
	return FileTask{
		Path:      urls.TransferSlash(fp),
		LocalPath: fp,
		IsDir:     info.IsDir,
		Sign:      info.Sign,
		Container: container,
		Modified:  info.Modified,
	}
}

// TaskWriter 将 openlist 文件写入到本地文件系统
type TaskWriter interface {

	// Path 将 openlist 文件路径中的文件名
	// 转换为本地文件系统中的文件名
	Path(path string) string

	// Write 将文件信息写入到本地文件系统中
	Write(task FileTask, localPath string) error
}

var (
	sw = StrmWriter{}
	rw = RawWriter{}
)

// LoadTaskWriter 根据文件容器加载 TaskWriter
func LoadTaskWriter(container string) TaskWriter {
	cfg := config.C.Openlist.LocalTreeGen
	if cfg.IsStrm(container) {
		return &sw
	}
	return &rw
}

// StrmWriter 写文件对应的 openlist strm 文件
type StrmWriter struct{}

// OpenlistPath 生成媒体的 openlist http 访问地址
func (sw *StrmWriter) OpenlistPath(task FileTask) string {
	segs := urls.Segments(task.Path)
	cfg := config.C.Openlist
	if cfg == nil {
		return ""
	}
	base := strings.TrimRight(cfg.Host, "/") + "/d"
	q := map[string]string{}
	if cfg.RequestWithSign != nil && *cfg.RequestWithSign && task.Sign != "" {
		q["sign"] = task.Sign
	}
	return urls.Build(base, segs, q)
}

// StrmContent 生成写入到 strm 文件内的内容
// 支持自定义路径前缀
func (sw *StrmWriter) StrmContent(task FileTask) string {
	rawSegs := urls.Segments(task.Path)
	cleanSegs := make([]string, 0, len(rawSegs))
	for _, seg := range rawSegs {
		if strings.Contains(seg, "\\'") {
			seg = strings.ReplaceAll(seg, "\\'", "'")
		}
		cleanSegs = append(cleanSegs, seg)
	}

	base := strings.TrimSpace(config.C.Openlist.LocalTreeGen.StrmContentBase)
	if base == "" {
		base = fmt.Sprintf("%s/d", strings.TrimRight(config.C.Openlist.Host, "/"))
	}
	base = strings.TrimRight(base, "/")
	return urls.Build(base, cleanSegs, nil)
}

// Path 将 openlist 文件路径中的文件名
// 转换为本地文件系统中的文件名
func (sw *StrmWriter) Path(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + ".strm"
}

// Write 将文件信息写入到本地文件系统中
func (sw *StrmWriter) Write(task FileTask, localPath string) error {
	if err := os.WriteFile(localPath, []byte(sw.StrmContent(task)), os.ModePerm); err != nil {
		return err
	}

	abs, err := filepath.Abs(localPath)
	if err != nil {
		abs = localPath
	}
	logf(colors.Gray, "生成 strm: [%s]", abs)

	return nil
}

// RawWriter 请求 openlist 源文件写入本地
type RawWriter struct {
	mu sync.Mutex
}

// Path 将 openlist 文件路径中的文件名
// 转换为本地文件系统中的文件名
func (rw *RawWriter) Path(path string) string {
	return path
}

// Write 将文件信息写入到本地文件系统中
func (rw *RawWriter) Write(task FileTask, localPath string) error {
	// 防止并发访问网盘触发风控
	rw.mu.Lock()
	defer rw.mu.Unlock()

	header := http.Header{"User-Agent": []string{constant.CommonDlUserAgent}}

	err := trys.Try(func() (err error) {
		logf(colors.Yellow, "尝试下载 openlist 源文件, 路径: [%s]", localPath)

		file, err := os.Create(localPath)
		if err != nil {
			return fmt.Errorf("创建文件失败 [%s]: %w", localPath, err)
		}
		defer file.Close()

		resp, err := https.Get(sw.OpenlistPath(task)).Header(header).Do()
		if err != nil {
			return fmt.Errorf("请求 openlist 直链失败: %w", err)
		}
		defer resp.Body.Close()

		if !https.IsSuccessCode(resp.StatusCode) {
			return fmt.Errorf("请求 openlist 直链失败, 响应状态: %s", resp.Status)
		}

		buf := bytess.CommonFixedBuffer()
		defer buf.PutBack()
		if _, err = io.CopyBuffer(file, resp.Body, buf.Bytes()); err != nil {
			return fmt.Errorf("写入 openlist 源文件到本地磁盘失败, 拷贝异常: %w", err)
		}

		logf(colors.Gray, "openlist 源文件 [%s] 已写入本地", filepath.Base(task.Path))
		return
	}, 3, time.Second*5)

	return err
}
