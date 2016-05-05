/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

// GenericReply generic rpc reply.
type GenericReply struct{}

// GenericArgs generic rpc args.
type GenericArgs struct{}

// ListVolsReply list vols rpc reply.
type ListVolsReply struct {
	Vols []VolInfo
}

// StatFileArgs stat file args.
type StatFileArgs struct {
	Vol  string
	Path string
}

// DeleteFileArgs delete file args.
type DeleteFileArgs struct {
	Vol  string
	Path string
}

// RenameFileArgs rename file args.
type RenameFileArgs struct {
	SrcVol  string
	SrcPath string
	DstVol  string
	DstPath string
}
