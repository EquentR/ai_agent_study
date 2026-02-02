package migrate

import (
	"agent_study/internal/db"
	"agent_study/internal/log"
	"agent_study/internal/model"
	"math"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

const (
	separator = "."
)

func AutoMigrate(currentVersion string, vm []Migration) {
	err := db.DB().AutoMigrate(&model.DataVersion{})
	if err != nil {
		log.Fatal("data version auto migrate failed", zap.Error(err))
	}

	savedVersion, err := GetDataVersion()
	if err != nil {
		log.Fatal("get data version failed", zap.Error(err))
	}

	//数据版本比应用版本还大
	if compareVersion(currentVersion, savedVersion) == -1 {
		log.Warn("the current version is smaller than the data version")
	}

	sort.Slice(vm, func(i, j int) bool {
		return compareVersion(vm[i].Version, vm[j].Version) == -1
	})

	for _, migration := range vm {
		if compareVersion(migration.Version, currentVersion) == 1 {
			//数据版本比当前版本还大，说明应用版本相对于数据版本回退了
			log.Warn("the migration version is greater than the application version")
		}
		//如果当前数据版本小于迁移版本，说明需要通过该过程升级数据
		if compareVersion(migration.Version, savedVersion) == 1 {
			migration.Fun()
			savedVersion = migration.Version
			SetDataVersion(db.DB(), savedVersion)
		}
	}

	if compareVersion(savedVersion, currentVersion) == -1 {
		//当前版本不是存储的数据版本，但是也不需要特殊数据升级的时候
		SetDataVersion(db.DB(), currentVersion)
	}
}

// compareVersion
// 对比 x.x.x这样版本的前后顺序
// 新的大是：-1
// 旧的大是： 1
// 相同是：   0
func compareVersion(old, new string) int {
	oldSplit := strings.Split(old, separator)
	newSplit := strings.Split(new, separator)
	max := int(math.Max(float64(len(oldSplit)), float64(len(newSplit))))
	for i := 0; i < max; i++ {
		oldV, err := getNum(oldSplit, i)
		if err != nil {
			log.Fatal("parse int failed", zap.Error(err))
		}
		newV, err := getNum(newSplit, i)
		if err != nil {
			log.Fatal("parse int failed", zap.Error(err))
		}
		if oldV < newV {
			return -1
		}
		if newV < oldV {
			return 1
		}
	}
	return 0
}

func getNum(s []string, i int) (int, error) {
	if i < len(s) {
		if s[i] == "" {
			return 0, nil
		}
		parseInt, err := strconv.ParseInt(s[i], 10, 64)
		if err != nil {
			return 0, err
		}
		return int(parseInt), err
	} else {
		return 0, nil
	}
}
