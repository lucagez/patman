.PHONY: all patman

patman: 
	@echo "Compiling Patman binary..."
	@mkdir -p build
	@go build -o build/patman cmd/patman/main.go
	@echo "Patman binary located in build directory."