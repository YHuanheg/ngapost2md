package config

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"

	"gopkg.in/ini.v1"
)

// 配置读写会出现在 CLI 启动、Server 启动、API 读取/保存等多个位置，
// 且 GetConfigAutoUpdate 会在配置项缺失时回写文件，用互斥锁避免并发写坏 config.ini
var configFileMu sync.Mutex

// 定义默认配置。使用 slice 保证顺序
var sectionList = []string{"config", "network", "post", "server"}

var defaultConfig = map[string][][3]string{
	"config": {
		{"version", "2.0.0", "请勿修改此项。配置文件中的注释项不应用于记录信息，软件加载时会自动覆盖并删除。"},
	},
	"network": {
		{"base_url", "https://bbs.nga.cn", "软件访问的 NGA 域名。默认值为 https://bbs.nga.cn。"},
		{"ua", "<;MODIFY_ME;>", "浏览器 User-Agent，填写你常用浏览器的 UA 即可。修改时请将尖括号及内部所有文本替换或删除，并确保值被反引号包裹。"},
		{"ngaPassportUid", "<;MODIFY_ME;>", "NGA 网站个人 Cookie 项目。修改时请将尖括号及内部所有文本替换或删除，并确保值被反引号包裹。"},
		{"ngaPassportCid", "<;MODIFY_ME;>", "NGA 网站个人 Cookie 项目。修改时请将尖括号及内部所有文本替换或删除，并确保值被反引号包裹。"},
		{"thread", "2", "网络并发数，理论上提高并发数可以增加下载速度。仅支持 1、2、3。若开启 enhance_ori_reply，请将此值设定为 1。默认值为 2。"},
		{"page_download_limit", "100", "[#56]每次下载限制新下载的大约页数。达到上限后需重新运行程序以继续下载，直至全部下载完成。允许范围为 -1（含）至 100（含）。当值为 0 或 -1 时则不限制。默认值为 100（约 100 页）。"},
	},
	"post": {
		{"get_ip_location", "False", "[#44]是否查询用户基于 IP 的地理位置？若启用则会导致网络请求量最高增加 20 倍。默认值为 False（不启用）。"},
		{"enhance_ori_reply", "False", "[#35]尝试将被引用回复的原始楼层内容补充完整。开启此功能要求同步将 thread 线程数设置为 1，且需全新拉取，否则可能会补充到未格式化的文本。默认值为 False（不启用）。"},
		{"enhance_ori_reply_online", "False", "[#122]启用在线模式以增强原始回复。开启此功能要求同步开启 enhance_ori_reply，开启后可能从网络获取被回复的楼层内容。默认值为 False（不启用）。"},
		{"use_local_smile_pic", "False", "[#58]是否使用本地表情图片资源，而不是引用在线资源。默认值为 False（不启用）。"},
		{"local_smile_pic_path", "../smile/", "[#58]本地表情图片资源路径。支持绝对路径与相对路径。路径末尾需包含 /。"},
		{"use_title_as_folder_name", "False", "[#21]文件夹名是否使用帖子标题。默认值为 False。修改后仅对全新拉取的 tid 生效。"},
		{"use_title_as_md_file_name", "False", "[#21]Markdown 文件名是否使用帖子标题。默认值为 False。修改后仅对全新拉取的 tid 生效。"},
		{"use_network_media_url", "False", "[#109]是否直接使用媒体文件的在线链接，而不是将媒体资源下载到本地后做本地引用。默认值为 False（不启用）。"},
		{"assets_path", "./assets/", "[#124]帖子媒体资源存储路径。留空或设为 '.' 时，资源将保存在与输出文件相同的目录下。默认值为 ./assets/。"},
		{"split_md_file", "-1", "[#105]是否切分生成的 md 文件，并指定切分后每个文件大约包含的页数。1 页约等于 20 层。允许范围为 -1（含）至 200（含）。当值为 0 或 -1 时，不进行切分。默认值为 -1（不切分）。"},
		{"output_path", "./", "帖子输出目录。支持绝对路径与相对路径。默认值为 ./（当前程序所在目录）。"},
	},
	"server": {
		{"host", "0.0.0.0", "Server 模式绑定 IP。可被 --host 参数覆盖。默认值为 0.0.0.0（所有网络接口）。"},
		{"password", "<;MODIFY_ME;>", "Server 模式的 Basic Auth 密码。若未设置密码，Server 模式将拒绝启动。"},
		{"port", "8080", "Server 模式默认监听端口。可被 --port 参数覆盖。"},
	},
}

// GeneratePassword 生成一个随机 12-15 位密码，包含大写字母、小写字母和数字。
func GeneratePassword() string {
	const (
		upper  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lower  = "abcdefghijklmnopqrstuvwxyz"
		digits = "0123456789"
		all    = upper + lower + digits
	)

	n, err := rand.Int(rand.Reader, big.NewInt(4))
	if err != nil {
		n = big.NewInt(2)
	}
	length := 12 + int(n.Int64())

	buf := make([]byte, length)
	for i := range buf {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		if err != nil {
			buf[i] = all[0]
			continue
		}
		buf[i] = all[idx.Int64()]
	}

	// 确保至少包含每种必需的字符类型
	upperIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(upper))))
	buf[0] = upper[upperIdx.Int64()]
	lowerIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(lower))))
	buf[1] = lower[lowerIdx.Int64()]
	digitIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
	buf[2] = digits[digitIdx.Int64()]

	// 打乱顺序
	for i := length - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			continue
		}
		buf[i], buf[j.Int64()] = buf[j.Int64()], buf[i]
	}

	return string(buf)
}

// 会自动更新、格式化配置文件并保存
func GetConfigAutoUpdate() (*ini.File, error) {
	configFileMu.Lock()
	defer configFileMu.Unlock()

	// 不要等号对齐，那样子好难看
	ini.PrettyFormat = false
	// 打开旧的INI配置文件
	cfg, err := ini.Load("config.ini")
	if err != nil {
		if os.IsNotExist(err) {
			// 首次运行（或用户只下载了可执行文件）时目录下没有 config.ini，
			// 直接生成一份完整的默认配置，用户无需再手工执行 --gen-config-file
			if saveErr := SaveDefaultConfigFile(); saveErr != nil {
				return nil, fmt.Errorf("未找到 config.ini，且生成默认配置文件失败: %v", saveErr)
			}
			return nil, fmt.Errorf("未找到 config.ini，已自动生成默认配置文件。请填写 ua、ngaPassportUid、ngaPassportCid 后重新运行")
		}
		return nil, fmt.Errorf("无法加载配置文件: %v", err)
	}

	if !cfg.Section("config").HasKey("version") {
		// 确保错误配置文件不会导致被覆盖
		return nil, fmt.Errorf("无法找到配置文件版本信息！可能读取错误，或配置文件已损坏。请手动重新填写默认配置文件。")
	}

	localCfgVersion := cfg.Section("config").Key("version").String()
	// 此为默认配置
	defaultcfg := genDefaultConfig()
	latestCfgVersion := defaultcfg.Section("config").Key("version").String()

	// 针对相同功能，新配置相比于旧配置名字不同，需要进行自动迁移
	if cfg.Section("post").HasKey("enable_post_title") && !cfg.Section("post").HasKey("use_title_as_md_file_name") {
		// 从1.2.0->1.4.0时，enable_post_title 更名为 use_title_as_md_file_name
		var oldValue string
		if cfg.Section("post").Key("enable_post_title").MustBool() {
			oldValue = "True"
		} else {
			oldValue = "False"
		}
		defaultcfg.Section("post").Key("use_title_as_md_file_name").SetValue(oldValue)
	}
	if cfg.Section("post").HasKey("use_network_pic_url") && !cfg.Section("post").HasKey("use_network_media_url") {
		// 从1.8.0/1.9.0->1.10.0，use_network_pic_url 更名为 use_network_media_url
		var oldValue string
		if cfg.Section("post").Key("use_network_pic_url").MustBool() {
			oldValue = "True"
		} else {
			oldValue = "False"
		}
		defaultcfg.Section("post").Key("use_network_media_url").SetValue(oldValue)
	}

	// 基于默认配置，往默认配置内填已存在配置的信息，同时记录默认配置里已有的键
	knownKeys := map[string]map[string]bool{}
	for _, section := range defaultcfg.Sections() {
		if knownKeys[section.Name()] == nil {
			knownKeys[section.Name()] = map[string]bool{}
		}
		for _, key := range section.Keys() {
			knownKeys[section.Name()][key.Name()] = true
			if section.Name() == "config" && key.Name() == "version" {
				// 版本号不读取旧的，换用新的
				continue
			}
			if cfg.HasSection(section.Name()) && cfg.Section(section.Name()).HasKey(key.Name()) {
				// 读取配置内有此项，将value值填入默认配置内
				cfgValue := cfg.Section(section.Name()).Key(key.Name()).Value()
				key.SetValue(cfgValue)
			}
		}
	}

	// 分发包里附带的 config.ini 与程序内置默认配置是两份需要人工同步的副本，
	// 一旦发版时漏同步（例如 2.0.0 漏掉了 output_path 与 host），
	// 只按版本号决定是否回写就会导致同版本下缺失的配置项永远补不上，
	// 用户既看不到也无法修改这些配置项。此处显式检测缺失项并在必要时回写。
	var missingKeys []string
	for _, section := range defaultcfg.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}
		for _, key := range section.Keys() {
			if !cfg.HasSection(section.Name()) || !cfg.Section(section.Name()).HasKey(key.Name()) {
				missingKeys = append(missingKeys, section.Name()+"."+key.Name())
			}
		}
	}

	// 回写时以默认配置为模板重建整个文件，因此需要先把用户自行新增的配置项搬回去，避免丢失用户数据
	for _, section := range cfg.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}
		var extraKeys []*ini.Key
		for _, key := range section.Keys() {
			if knownKeys[section.Name()] != nil && knownKeys[section.Name()][key.Name()] {
				continue
			}
			extraKeys = append(extraKeys, key)
		}
		if len(extraKeys) == 0 {
			continue
		}
		targetSection, sectionErr := defaultcfg.GetSection(section.Name())
		if sectionErr != nil {
			targetSection, _ = defaultcfg.NewSection(section.Name())
		}
		if targetSection == nil {
			continue
		}
		for _, key := range extraKeys {
			targetSection.NewKey(key.Name(), key.Value())
		}
	}

	switch {
	case localCfgVersion != latestCfgVersion:
		if err = defaultcfg.SaveTo("config.ini"); err != nil {
			return nil, fmt.Errorf("无法保存更新后的配置文件: %v", err)
		}
		log.Println("配置文件已由", localCfgVersion, "自动更新至", latestCfgVersion, "，请查看引入的新功能特性。部分注释可能被移除或更改。")
	case len(missingKeys) > 0:
		// 版本号相同但缺少配置项（分发包内附带的 config.ini 落后于程序），同样需要补齐
		if err = defaultcfg.SaveTo("config.ini"); err != nil {
			return nil, fmt.Errorf("无法保存补全后的配置文件: %v", err)
		}
		log.Println("配置文件缺少以下配置项，已自动补全：", strings.Join(missingKeys, "、"))
	}
	return defaultcfg, nil
}

func genDefaultConfig() *ini.File {
	// 不要等号对齐，那样子好难看
	ini.PrettyFormat = false
	cfg := ini.Empty()
	for _, section := range sectionList {
		sectionNode, _ := cfg.NewSection(section)
		sectionDetail := defaultConfig[section]
		for _, v := range sectionDetail {
			kNode, _ := sectionNode.NewKey(v[0], v[1])
			kNode.Comment = v[2]
		}
	}
	return cfg
}

func SaveDefaultConfigFile() error {
	return genDefaultConfig().SaveTo("config.ini")
}

// SaveConfigFile 加锁地把配置写回 config.ini，避免与 GetConfigAutoUpdate 的回写并发冲突
func SaveConfigFile(cfg *ini.File) error {
	configFileMu.Lock()
	defer configFileMu.Unlock()
	return cfg.SaveTo("config.ini")
}
