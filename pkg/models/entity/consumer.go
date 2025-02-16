package entity

type RmqExchQueueInfo struct {
	Name       string `json:"name,omitempty"`
	Type       string `json:"type,omitempty"`
	AutoDelete bool   `json:"auto_delete,omitempty"`
	Durable    bool   `json:"durable,omitempty"`
	Internal   bool   `json:"internal,omitempty"`
	Exclusive  bool   `json:"exclusive,omitempty"`
	Nowait     bool   `json:"nowait,omitempty"`
	RoutingKey string `json:"routing_key,omitempty"`
	BatchAck   bool   `json:"batch_ack,omitempty"`
}

type RmqInputConf struct {
	Exch     *RmqExchQueueInfo `json:"exch,omitempty"`
	Queue    *RmqExchQueueInfo `json:"queue,omitempty"`
	BatchAck bool              `json:"batch_ack,omitempty"`
	Mode     string            `json:"mode,omitempty"`
}

type RmqOutputConf struct {
	Exch       *RmqExchQueueInfo    `json:"exch,omitempty"`
	Outputdefs []*RmqOutRoutingConf `json:"outputdefs,omitempty"`
	Mode       string               `json:"mode,omitempty"`
}

type RmqOutRoutingConf struct {
	RoutingKey string `json:"routing_key,omitempty"`
	Type       string `json:"type,omitempty"`
}

type ConsumerGroup struct {
	Name string `json:"name,omitempty"`
}

type DataManifestLabelConsumer struct {
	Ids []int64 `json:"ids"`
	IsShippo bool `json:"is_shippo"`
}
