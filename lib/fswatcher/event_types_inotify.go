// Copyright (C) 2017 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

// +build linux

package fswatcher

import "github.com/zillode/notify"

func (watcher *fsWatcher) eventMask() notify.Event {
	events := notify.InCreate | notify.InMovedTo | notify.InDelete |
		notify.InDeleteSelf | notify.InModify | notify.InMovedFrom | notify.InMoveSelf
	if !watcher.ignorePerms {
		events |= notify.InAttrib
	}
	return events
}

const removeEventMask = notify.InDelete | notify.InDeleteSelf |
	notify.InMovedFrom | notify.InMoveSelf
