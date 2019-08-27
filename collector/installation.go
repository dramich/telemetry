package collector

import (
	"regexp"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	UID_SETTING            = "telemetry-uid"
	SERVER_IMAGE_SETTING   = "server-image"
	SERVER_VERSION_SETTING = "server-version"
)

type Installation struct {
	Uid                  string     `json:"uid"`
	Image                string     `json:"image"`
	Version              string     `json:"version"`
	AuthConfig           LabelCount `json:"auth"`
	KontainerDriverCount int        `json:"kontainerDriverCount"`
	KontainerDrivers     LabelCount `json:"kontainerDrivers"`
	NodeDriverCount      int        `json:"nodeDriverCount"`
	NodeDrivers          LabelCount `json:"nodeDrivers"`
	HasInternal          bool       `json:"hasInternal"`
}

func (i Installation) RecordKey() string {
	return "install"
}

func (i Installation) Collect(c *CollectorOpts) interface{} {
	log.Debug("Collecting Installation")

	nonRemoved := NonRemoved()

	settings := GetSettingCollection(c.Client)

	uid, _ := GetSettingByCollection(settings, UID_SETTING)
	uid, _ = i.GetUid(uid, c)

	i.Uid = uid
	i.Image = "unknown"
	i.Version = "unknown"
	i.AuthConfig = make(LabelCount)
	i.KontainerDrivers = make(LabelCount)
	i.NodeDrivers = make(LabelCount)

	if image, ok := GetSettingByCollection(settings, SERVER_IMAGE_SETTING); ok {
		log.Debugf("  Image: %s", image)
		if image != "" {
			i.Image = image
		}
	}

	if version, ok := GetSettingByCollection(settings, SERVER_VERSION_SETTING); ok {
		log.Debugf("  Version: %s", version)
		if version != "" {
			i.Version = version
		}
	}

	log.Debug("Collecting AuthConfigs")
	configList, err := c.Client.AuthConfig.List(&nonRemoved)
	if err == nil {
		for _, config := range configList.Data {
			if config.Enabled {
				name := regexp.MustCompile("(?i)^(.*?)Config$").ReplaceAllString(config.Type, "$1")
				i.AuthConfig.Increment(name)
			}
		}
	} else {
		log.Errorf("Failed to get authProviders err=%s", err)
	}

	log.Debug("Collecting NodeDrivers")
	nodeDriverList, err := c.Client.NodeDriver.List(&nonRemoved)
	if err == nil {
		for _, driver := range nodeDriverList.Data {
			if driver.Active {
				i.NodeDrivers.Increment(driver.Name)
				i.NodeDriverCount++
			}
		}
	} else {
		log.Errorf("Failed to get nodeDrivers err=%s", err)
	}

	log.Debug("Collecting KontainerDrivers")
	kontainerDriverList, err := c.Client.KontainerDriver.List(&nonRemoved)
	if err == nil {
		for _, driver := range kontainerDriverList.Data {
			if driver.Active {
				i.KontainerDrivers.Increment(driver.Name)
				i.KontainerDriverCount++
			}
		}
	} else {
		log.Errorf("Failed to get kontainerDrivers err=%s", err)
	}

	i.HasInternal = false

	log.Debug("Looking for Local cluser")
	clusterList, err := c.Client.Cluster.List(&nonRemoved)
	if err == nil {
		for _, cluster := range clusterList.Data {
			if cluster.Internal {
				i.HasInternal = true
				break
			}
		}
	} else {
		log.Errorf("Failed to get Clusters err=%s", err)
	}

	return i
}

func (i Installation) GetUid(uid string, c *CollectorOpts) (string, bool) {
	if uid != "" {
		log.Debugf("  Using Existing Uid: %s", uid)
		return uid, true
	}

	newuid, _ := uuid.NewV4()
	uid = newuid.String()
	err := SetSetting(c.Client, UID_SETTING, uid)
	if err != nil {
		log.Debugf("  Error Generating Uid: %s", err)
		return "", false
	}

	log.Debugf("  Generated Uid: %s", uid)
	return uid, true
}

func init() {
	Register(Installation{})
}
