package config

import (
	"errors"
	"strings"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/constant"
)

type Navidrome struct {
	Host string `yaml:"host"`
}

func (n *Navidrome) Init() error {
	if strings.TrimSpace(constant.ServerMode) == "navidrome" {
		if strings.TrimSpace(n.Host) == "" {
			return errors.New("navidrome.host 配置不能为空")
		}
	}
	return nil
}
