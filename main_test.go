package main

import (
	"testing"

	"github.com/bitrise-steplib/steps-cache-push/model"
)

func Test_isSameStack(t *testing.T) {
	tests := []struct {
		name         string
		archiveStack model.ArchiveInfo
		currentStack model.ArchiveInfo
		want         bool
	}{
		{
			name: "Going from empty to iOS",
			archiveStack: model.ArchiveInfo{
				StackID: "",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: false,
		},
		{
			name: "Going from iOS to empty",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: model.ArchiveInfo{
				StackID: "",
			},
			want: false,
		},
		{
			name: "Going from Gen2 to Gen1",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: true,
		},
		{
			name: "Going from Gen2 to Gen2 same machine",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: true,
		},
		{
			name: "Going from Gen2 to Gen2 different stack",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.4.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: false,
		},
		{
			name: "Going from Gen2 to Gen2 different machine",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-4c-20gb-300gb-atl01-ded001",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: true,
		},
		{
			name: "Going from Gen1 to Gen2",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: true,
		},
		{
			name: "Going from Gen1 to Gen2 different stack",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.4.x",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: false,
		},
		{
			name: "Going from Gen2 to Gen1 different stack",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.4.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu to iOS",
			archiveStack: model.ArchiveInfo{
				StackID: "linux-docker-android",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu to Ubuntu",
			archiveStack: model.ArchiveInfo{
				StackID: "linux-docker-android",
			},
			currentStack: model.ArchiveInfo{
				StackID: "linux-docker-android",
			},
			want: true,
		},
		{
			name: "Going from iOS to Ubuntu",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: model.ArchiveInfo{
				StackID: "linux-docker-android",
			},
			want: false,
		},
		{
			name: "Going from iOS to iOS same stack",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: true,
		},
		{
			name: "Going from iOS to iOS different stack",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.4.x",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu to Ubuntu LTS",
			archiveStack: model.ArchiveInfo{
				StackID: "linux-docker-android",
			},
			currentStack: model.ArchiveInfo{
				StackID: "linux-docker-android-lts",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu LTS to Ubuntu",
			archiveStack: model.ArchiveInfo{
				StackID: "linux-docker-android-lts",
			},
			currentStack: model.ArchiveInfo{
				StackID: "linux-docker-android",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu to Gen2 iOS",
			archiveStack: model.ArchiveInfo{
				StackID: "linux-docker-android",
			},
			currentStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: false,
		},
		{
			name: "Going from iOS to iOS, architecture introduced",
			archiveStack: model.ArchiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: model.ArchiveInfo{
				StackID:      "osx-xcode-12.3.x",
				Architecture: "amd64",
			},
			want: true,
		},
		{
			name: "Going from iOS to iOS, same id, same arch",
			archiveStack: model.ArchiveInfo{
				StackID:      "osx-xcode-12.3.x",
				Architecture: "amd64",
			},
			currentStack: model.ArchiveInfo{
				StackID:      "osx-xcode-12.3.x",
				Architecture: "amd64",
			},
			want: true,
		},
		{
			name: "Going from iOS to iOS, same id, different arch, ignore version",
			archiveStack: model.ArchiveInfo{
				Version:      1,
				StackID:      "osx-xcode-12.3.x",
				Architecture: "amd64",
			},
			currentStack: model.ArchiveInfo{
				Version:      2,
				StackID:      "osx-xcode-12.3.x",
				Architecture: "arm64",
			},
			want: false,
		},
		{
			name: "Going from iOS to iOS, different id, same arch",
			archiveStack: model.ArchiveInfo{
				StackID:      "osx-xcode-12.3.x",
				Architecture: "arm64",
			},
			currentStack: model.ArchiveInfo{
				StackID:      "osx-xcode-12.4.x",
				Architecture: "arm64",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSameStack(tt.archiveStack, tt.currentStack); got != tt.want {
				t.Errorf("isSameStack() = %v, want %v", got, tt.want)
			}
		})
	}
}
