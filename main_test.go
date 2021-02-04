package main

import "testing"

func Test_isSameStack(t *testing.T) {
	type args struct {
		archiveStackID string
		currentStackID string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Going from empty to iOS",
			args{archiveStackID: "", currentStackID: "osx-xcode-12.3.x"},
			false,
		},
		{
			"Going from iOS to empty",
			args{archiveStackID: "osx-xcode-12.3.x", currentStackID: ""},
			false,
		},
		{
			"Going from Gen2 to Gen1",
			args{archiveStackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001", currentStackID: "osx-xcode-12.3.x"},
			true,
		},
		{
			"Going from Gen2 to Gen2 same machine",
			args{archiveStackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001", currentStackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001"},
			true,
		},
		{
			"Going from Gen2 to Gen2 different stack",
			args{archiveStackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001", currentStackID: "osx-xcode-12.4.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001"},
			false,
		},
		{
			"Going from Gen2 to Gen2 different machine",
			args{archiveStackID: "osx-xcode-12.3.x-gen2-mmg4-4c-20gb-300gb-atl01-ded001", currentStackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001"},
			true,
		},
		{
			"Going from Gen1 to Gen2",
			args{archiveStackID: "osx-xcode-12.3.x", currentStackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001"},
			true,
		},
		{
			"Going from Gen1 to Gen2 different stack",
			args{archiveStackID: "osx-xcode-12.4.x", currentStackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001"},
			false,
		},
		{
			"Going from Gen2 to Gen1 different stack",
			args{archiveStackID: "osx-xcode-12.4.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001", currentStackID: "osx-xcode-12.3.x"},
			false,
		},
		{
			"Going from Ubuntu to iOS",
			args{archiveStackID: "linux-docker-android", currentStackID: "osx-xcode-12.3.x"},
			false,
		},
		{
			"Going from Ubuntu to Ubuntu",
			args{archiveStackID: "linux-docker-android", currentStackID: "linux-docker-android"},
			true,
		},
		{
			"Going from iOS to Ubuntu",
			args{archiveStackID: "osx-xcode-12.3.x", currentStackID: "linux-docker-android"},
			false,
		},
		{
			"Going from iOS to iOS same stack",
			args{archiveStackID: "osx-xcode-12.3.x", currentStackID: "osx-xcode-12.3.x"},
			true,
		},
		{
			"Going from iOS to iOS different stack",
			args{archiveStackID: "osx-xcode-12.3.x", currentStackID: "osx-xcode-12.4.x"},
			false,
		},
		{
			"Going from Ubuntu to Ubuntu LTS",
			args{archiveStackID: "linux-docker-android", currentStackID: "linux-docker-android-lts"},
			false,
		},
		{
			"Going from Ubuntu LTS to Ubuntu",
			args{archiveStackID: "linux-docker-android-lts", currentStackID: "linux-docker-android"},
			false,
		},
		{
			"Going from Ubuntu to Gen2 iOS",
			args{archiveStackID: "linux-docker-android", currentStackID: "osx-xcode-12.3.x-gen2-mmg4-12c-60gb-300gb-atl01-ded001"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSameStack(tt.args.archiveStackID, tt.args.currentStackID); got != tt.want {
				t.Errorf("isSameStack() = %v, want %v", got, tt.want)
			}
		})
	}
}
