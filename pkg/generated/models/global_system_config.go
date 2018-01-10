package models

// GlobalSystemConfig

import "encoding/json"

// GlobalSystemConfig
type GlobalSystemConfig struct {
	BGPAlwaysCompareMed       bool                           `json:"bgp_always_compare_med"`
	AutonomousSystem          AutonomousSystemType           `json:"autonomous_system"`
	DisplayName               string                         `json:"display_name"`
	ConfigVersion             string                         `json:"config_version"`
	BgpaasParameters          *BGPaaServiceParametersType    `json:"bgpaas_parameters"`
	MacAgingTime              MACAgingTime                   `json:"mac_aging_time"`
	FQName                    []string                       `json:"fq_name"`
	UUID                      string                         `json:"uuid"`
	MacMoveControl            *MACMoveLimitControlType       `json:"mac_move_control"`
	PluginTuning              *PluginProperties              `json:"plugin_tuning"`
	MacLimitControl           *MACLimitControlType           `json:"mac_limit_control"`
	ParentType                string                         `json:"parent_type"`
	Annotations               *KeyValuePairs                 `json:"annotations"`
	Perms2                    *PermType2                     `json:"perms2"`
	AlarmEnable               bool                           `json:"alarm_enable"`
	IbgpAutoMesh              bool                           `json:"ibgp_auto_mesh"`
	IPFabricSubnets           *SubnetListType                `json:"ip_fabric_subnets"`
	ParentUUID                string                         `json:"parent_uuid"`
	IDPerms                   *IdPermsType                   `json:"id_perms"`
	UserDefinedLogStatistics  *UserDefinedLogStatList        `json:"user_defined_log_statistics"`
	GracefulRestartParameters *GracefulRestartParametersType `json:"graceful_restart_parameters"`

	BGPRouterRefs []*GlobalSystemConfigBGPRouterRef `json:"bgp_router_refs"`

	Alarms               []*Alarm               `json:"alarms"`
	AnalyticsNodes       []*AnalyticsNode       `json:"analytics_nodes"`
	APIAccessLists       []*APIAccessList       `json:"api_access_lists"`
	ConfigNodes          []*ConfigNode          `json:"config_nodes"`
	DatabaseNodes        []*DatabaseNode        `json:"database_nodes"`
	GlobalQosConfigs     []*GlobalQosConfig     `json:"global_qos_configs"`
	GlobalVrouterConfigs []*GlobalVrouterConfig `json:"global_vrouter_configs"`
	PhysicalRouters      []*PhysicalRouter      `json:"physical_routers"`
	ServiceApplianceSets []*ServiceApplianceSet `json:"service_appliance_sets"`
	VirtualRouters       []*VirtualRouter       `json:"virtual_routers"`
}

// GlobalSystemConfigBGPRouterRef references each other
type GlobalSystemConfigBGPRouterRef struct {
	UUID string   `json:"uuid"`
	To   []string `json:"to"` //FQDN

}

// String returns json representation of the object
func (model *GlobalSystemConfig) String() string {
	b, _ := json.Marshal(model)
	return string(b)
}

// MakeGlobalSystemConfig makes GlobalSystemConfig
func MakeGlobalSystemConfig() *GlobalSystemConfig {
	return &GlobalSystemConfig{
		//TODO(nati): Apply default
		MacAgingTime:              MakeMACAgingTime(),
		FQName:                    []string{},
		UUID:                      "",
		MacMoveControl:            MakeMACMoveLimitControlType(),
		PluginTuning:              MakePluginProperties(),
		MacLimitControl:           MakeMACLimitControlType(),
		ParentType:                "",
		Annotations:               MakeKeyValuePairs(),
		Perms2:                    MakePermType2(),
		AlarmEnable:               false,
		IbgpAutoMesh:              false,
		IPFabricSubnets:           MakeSubnetListType(),
		ParentUUID:                "",
		IDPerms:                   MakeIdPermsType(),
		UserDefinedLogStatistics:  MakeUserDefinedLogStatList(),
		GracefulRestartParameters: MakeGracefulRestartParametersType(),
		BGPAlwaysCompareMed:       false,
		AutonomousSystem:          MakeAutonomousSystemType(),
		DisplayName:               "",
		ConfigVersion:             "",
		BgpaasParameters:          MakeBGPaaServiceParametersType(),
	}
}

// InterfaceToGlobalSystemConfig makes GlobalSystemConfig from interface
func InterfaceToGlobalSystemConfig(iData interface{}) *GlobalSystemConfig {
	data := iData.(map[string]interface{})
	return &GlobalSystemConfig{
		MacMoveControl: InterfaceToMACMoveLimitControlType(data["mac_move_control"]),

		//{"description":"MAC move control on the network","type":"object","properties":{"mac_move_limit":{"type":"integer"},"mac_move_limit_action":{"type":"string","enum":["log","alarm","shutdown","drop"]},"mac_move_time_window":{"type":"integer","minimum":1,"maximum":60}}}
		PluginTuning: InterfaceToPluginProperties(data["plugin_tuning"]),

		//{"description":"Various Orchestration system plugin(interface) parameters, like Openstack Neutron plugin.","type":"object","properties":{"plugin_property":{"type":"array","item":{"type":"object","properties":{"property":{"type":"string"},"value":{"type":"string"}}}}}}
		MacAgingTime: InterfaceToMACAgingTime(data["mac_aging_time"]),

		//{"description":"MAC aging time on the network","default":"300","type":"integer","minimum":0,"maximum":86400}
		FQName: data["fq_name"].([]string),

		//{"type":"array","item":{"type":"string"}}
		UUID: data["uuid"].(string),

		//{"type":"string"}
		Perms2: InterfaceToPermType2(data["perms2"]),

		//{"type":"object","properties":{"global_access":{"type":"integer","minimum":0,"maximum":7},"owner":{"type":"string"},"owner_access":{"type":"integer","minimum":0,"maximum":7},"share":{"type":"array","item":{"type":"object","properties":{"tenant":{"type":"string"},"tenant_access":{"type":"integer","minimum":0,"maximum":7}}}}}}
		AlarmEnable: data["alarm_enable"].(bool),

		//{"description":"Flag to enable/disable alarms configured under global-system-config. True, if not set.","type":"boolean"}
		IbgpAutoMesh: data["ibgp_auto_mesh"].(bool),

		//{"description":"When true, system will automatically create BGP peering mesh with all control-nodes that have same BGP AS number as global AS number.","type":"boolean"}
		MacLimitControl: InterfaceToMACLimitControlType(data["mac_limit_control"]),

		//{"description":"MAC limit control on the network","type":"object","properties":{"mac_limit":{"type":"integer"},"mac_limit_action":{"type":"string","enum":["log","alarm","shutdown","drop"]}}}
		ParentType: data["parent_type"].(string),

		//{"type":"string"}
		Annotations: InterfaceToKeyValuePairs(data["annotations"]),

		//{"type":"object","properties":{"key_value_pair":{"type":"array","item":{"type":"object","properties":{"key":{"type":"string"},"value":{"type":"string"}}}}}}
		UserDefinedLogStatistics: InterfaceToUserDefinedLogStatList(data["user_defined_log_statistics"]),

		//{"description":"stats name and patterns","type":"object","properties":{"statlist":{"type":"array","item":{"type":"object","properties":{"name":{"type":"string"},"pattern":{"type":"string"}}}}}}
		GracefulRestartParameters: InterfaceToGracefulRestartParametersType(data["graceful_restart_parameters"]),

		//{"description":"Graceful Restart parameters","type":"object","properties":{"bgp_helper_enable":{"type":"boolean"},"enable":{"type":"boolean"},"end_of_rib_timeout":{"type":"integer","minimum":0,"maximum":4095},"long_lived_restart_time":{"type":"integer","minimum":0,"maximum":16777215},"restart_time":{"type":"integer","minimum":0,"maximum":4095},"xmpp_helper_enable":{"type":"boolean"}}}
		IPFabricSubnets: InterfaceToSubnetListType(data["ip_fabric_subnets"]),

		//{"description":"List of all subnets in which vrouter ip address exist. Used by Device manager to configure dynamic GRE tunnels on the SDN gateway.","type":"object","properties":{"subnet":{"type":"array","item":{"type":"object","properties":{"ip_prefix":{"type":"string"},"ip_prefix_len":{"type":"integer"}}}}}}
		ParentUUID: data["parent_uuid"].(string),

		//{"type":"string"}
		IDPerms: InterfaceToIdPermsType(data["id_perms"]),

		//{"type":"object","properties":{"created":{"type":"string"},"creator":{"type":"string"},"description":{"type":"string"},"enable":{"type":"boolean"},"last_modified":{"type":"string"},"permissions":{"type":"object","properties":{"group":{"type":"string"},"group_access":{"type":"integer","minimum":0,"maximum":7},"other_access":{"type":"integer","minimum":0,"maximum":7},"owner":{"type":"string"},"owner_access":{"type":"integer","minimum":0,"maximum":7}}},"user_visible":{"type":"boolean"}}}
		ConfigVersion: data["config_version"].(string),

		//{"description":"Version of OpenContrail software that generated this config.","type":"string"}
		BgpaasParameters: InterfaceToBGPaaServiceParametersType(data["bgpaas_parameters"]),

		//{"description":"BGP As A Service Parameters configuration","type":"object","properties":{"port_end":{"type":"integer","minimum":-1,"maximum":65535},"port_start":{"type":"integer","minimum":-1,"maximum":65535}}}
		BGPAlwaysCompareMed: data["bgp_always_compare_med"].(bool),

		//{"description":"Always compare MED even if paths are received from different ASes.","type":"boolean"}
		AutonomousSystem: InterfaceToAutonomousSystemType(data["autonomous_system"]),

		//{"description":"16 bit BGP Autonomous System number for the cluster.","type":"integer","minimum":1,"maximum":65534}
		DisplayName: data["display_name"].(string),

		//{"type":"string"}

	}
}

// InterfaceToGlobalSystemConfigSlice makes a slice of GlobalSystemConfig from interface
func InterfaceToGlobalSystemConfigSlice(data interface{}) []*GlobalSystemConfig {
	list := data.([]interface{})
	result := MakeGlobalSystemConfigSlice()
	for _, item := range list {
		result = append(result, InterfaceToGlobalSystemConfig(item))
	}
	return result
}

// MakeGlobalSystemConfigSlice() makes a slice of GlobalSystemConfig
func MakeGlobalSystemConfigSlice() []*GlobalSystemConfig {
	return []*GlobalSystemConfig{}
}
