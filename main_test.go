package main

import "testing"

func Test_isSameStack(t *testing.T) {
	tests := []struct {
		name         string
		archiveStack archiveInfo
		currentStack archiveInfo
		want         bool
	}{
		{
			name: "Going from empty to iOS",
			archiveStack: archiveInfo{
				StackID: "",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: false,
		},
		{
			name: "Going from iOS to empty",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: archiveInfo{
				StackID: "",
			},
			want: false,
		},
		{
			name: "Going from Gen2 to Gen1",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: true,
		},
		{
			name: "Going from Gen2 to Gen2 same machine",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: true,
		},
		{
			name: "Going from Gen2 to Gen2 different stack",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.4.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: false,
		},
		{
			name: "Going from Gen2 to Gen2 different machine",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-4c-20gb-300gb-atl01-ded001",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: true,
		},
		{
			name: "Going from Gen1 to Gen2",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: true,
		},
		{
			name: "Going from Gen1 to Gen2 different stack",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.4.x",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: false,
		},
		{
			name: "Going from Gen2 to Gen1 different stack",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.4.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu to iOS",
			archiveStack: archiveInfo{
				StackID: "linux-docker-android",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu to Ubuntu",
			archiveStack: archiveInfo{
				StackID: "linux-docker-android",
			},
			currentStack: archiveInfo{
				StackID: "linux-docker-android",
			},
			want: true,
		},
		{
			name: "Going from iOS to Ubuntu",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: archiveInfo{
				StackID: "linux-docker-android",
			},
			want: false,
		},
		{
			name: "Going from iOS to iOS same stack",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			want: true,
		},
		{
			name: "Going from iOS to iOS different stack",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.4.x",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu to Ubuntu LTS",
			archiveStack: archiveInfo{
				StackID: "linux-docker-android",
			},
			currentStack: archiveInfo{
				StackID: "linux-docker-android-lts",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu LTS to Ubuntu",
			archiveStack: archiveInfo{
				StackID: "linux-docker-android-lts",
			},
			currentStack: archiveInfo{
				StackID: "linux-docker-android",
			},
			want: false,
		},
		{
			name: "Going from Ubuntu to Gen2 iOS",
			archiveStack: archiveInfo{
				StackID: "linux-docker-android",
			},
			currentStack: archiveInfo{
				StackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001",
			},
			want: false,
		},
		{
			name: "Going from iOS to iOS, architecture introduced",
			archiveStack: archiveInfo{
				StackID: "osx-xcode-12.3.x",
			},
			currentStack: archiveInfo{
				StackID:     "osx-xcode-12.3.x",
				Arhitecture: "amd64",
			},
			want: false,
		},
		{
			name: "Going from iOS to iOS, same id, same arch",
			archiveStack: archiveInfo{
				StackID:     "osx-xcode-12.3.x",
				Arhitecture: "amd64",
			},
			currentStack: archiveInfo{
				StackID:     "osx-xcode-12.3.x",
				Arhitecture: "amd64",
			},
			want: true,
		},
		{
			name: "Going from iOS to iOS, same id, different arch",
			archiveStack: archiveInfo{
				StackID:     "osx-xcode-12.3.x",
				Arhitecture: "amd64",
			},
			currentStack: archiveInfo{
				StackID:     "osx-xcode-12.3.x",
				Arhitecture: "arm64",
			},
			want: false,
		},
		{
			name: "Going from iOS to iOS, different id, same arch",
			archiveStack: archiveInfo{
				StackID:     "osx-xcode-12.3.x",
				Arhitecture: "arm64",
			},
			currentStack: archiveInfo{
				StackID:     "osx-xcode-12.4.x",
				Arhitecture: "arm64",
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
