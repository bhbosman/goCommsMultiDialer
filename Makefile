all:
	mockgen -package netMultiDialer -generateWhat ddd -destination INetMultiDialerInterfaceMethods.go . INetMultiDialer
