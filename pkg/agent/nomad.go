package agent

type ConnectRequest struct {
	// service addresses I'd like to connect to
	desiredServices []string

	taskID TaskID
}

type Allocation struct {
	ID            string
	TaskResources map[string]TaskResource
}

type TaskResource struct {
	Name      string
	IPAddress string
	Ports     []ResourcePort
}

type ResourcePort struct {
	// Value is the port the service is actually broascasting on
	Value int
	// Listening is the value the port the container thinks it's listening on
	Listening int
}
