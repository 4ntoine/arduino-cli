// This file is part of arduino-cli.
//
// Copyright 2020 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the GNU General Public License version 3,
// which covers the main part of arduino-cli.
// The terms of this license can be found at:
// https://www.gnu.org/licenses/gpl-3.0.en.html
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package core

import (
	"context"

	"github.com/arduino/arduino-cli/arduino/cores"
	"github.com/arduino/arduino-cli/arduino/cores/packagemanager"
	"github.com/arduino/arduino-cli/commands"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
)

// PlatformUninstall FIXMEDOC
func PlatformUninstall(ctx context.Context, req *rpc.PlatformUninstallRequest, taskCB commands.TaskProgressCB) (*rpc.PlatformUninstallResponse, error) {
	pm := commands.GetPackageManager(req.GetInstance().GetId())
	if pm == nil {
		return nil, &commands.InvalidInstanceError{}
	}

	ref := &packagemanager.PlatformReference{
		Package:              req.PlatformPackage,
		PlatformArchitecture: req.Architecture,
	}
	if ref.PlatformVersion == nil {
		platform := pm.FindPlatform(ref)
		if platform == nil {
			return nil, &commands.PlatformNotFound{Platform: ref.String()}
		}
		platformRelease := pm.GetInstalledPlatformRelease(platform)
		if platformRelease == nil {
			return nil, &commands.PlatformNotFound{Platform: ref.String()}
		}
		ref.PlatformVersion = platformRelease.Version
	}

	platform, tools, err := pm.FindPlatformReleaseDependencies(ref)
	if err != nil {
		return nil, &commands.NotFoundError{Message: tr("Can't find dependencies for platform %s", ref), Cause: err}
	}

	if err := uninstallPlatformRelease(pm, platform, taskCB); err != nil {
		return nil, err
	}

	for _, tool := range tools {
		if !pm.IsToolRequired(tool) {
			uninstallToolRelease(pm, tool, taskCB)
		}
	}

	if err := commands.Init(&rpc.InitRequest{Instance: req.Instance}, nil); err != nil {
		return nil, err
	}

	return &rpc.PlatformUninstallResponse{}, nil
}

func uninstallPlatformRelease(pm *packagemanager.PackageManager, platformRelease *cores.PlatformRelease, taskCB commands.TaskProgressCB) error {
	log := pm.Log.WithField("platform", platformRelease)

	log.Info("Uninstalling platform")
	taskCB(&rpc.TaskProgress{Name: tr("Uninstalling %s", platformRelease)})

	if err := pm.UninstallPlatform(platformRelease); err != nil {
		log.WithError(err).Error("Error uninstalling")
		return &commands.FailedUninstallError{Message: tr("Error uninstalling platform %s", platformRelease), Cause: err}
	}

	log.Info("Platform uninstalled")
	taskCB(&rpc.TaskProgress{Message: tr("Platform %s uninstalled", platformRelease), Completed: true})
	return nil
}

func uninstallToolRelease(pm *packagemanager.PackageManager, toolRelease *cores.ToolRelease, taskCB commands.TaskProgressCB) error {
	log := pm.Log.WithField("Tool", toolRelease)

	log.Info("Uninstalling tool")
	taskCB(&rpc.TaskProgress{Name: tr("Uninstalling %s, tool is no more required", toolRelease)})

	if err := pm.UninstallTool(toolRelease); err != nil {
		log.WithError(err).Error("Error uninstalling")
		return &commands.FailedUninstallError{Message: tr("Error uninstalling tool %s", toolRelease), Cause: err}
	}

	log.Info("Tool uninstalled")
	taskCB(&rpc.TaskProgress{Message: tr("Tool %s uninstalled", toolRelease), Completed: true})
	return nil
}
