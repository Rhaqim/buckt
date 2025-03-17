package web

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type RouterInitialiser func(bucktLog *logger.BucktLogger, standaloneMode, debug bool, fileService domain.FileService, folderService domain.FolderService) (domain.RouterService, error)

var InitialisedRouterService RouterInitialiser

func RegisterRouterInitialiser(f RouterInitialiser) {
	InitialisedRouterService = f
}

func GetRouterService(bucktLog *logger.BucktLogger, standaloneMode, debug bool, fileService domain.FileService, folderService domain.FolderService) (domain.RouterService, error) {
	return InitialisedRouterService(bucktLog, standaloneMode, debug, fileService, folderService)
}
