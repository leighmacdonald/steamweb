# steamweb

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build](https://github.com/leighmacdonald/steamweb/actions/workflows/check.yml/badge.svg?branch=master)](https://github.com/leighmacdonald/steamweb/actions/workflows/check.yml)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f06234b0551a49cc8ac111d7b77827b2)](https://www.codacy.com/manual/leighmacdonald/steamweb?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=leighmacdonald/steamweb&amp;utm_campaign=Badge_Grade)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmacdonald/steamweb)](https://goreportcard.com/report/github.com/leighmacdonald/steamweb)
[![GoDoc](https://godoc.org/github.com/leighmacdonald/steamweb?status.svg)](https://pkg.go.dev/github.com/leighmacdonald/steamweb)
[![Discord chat](https://img.shields.io/discord/704508824320475218)](https://discord.gg/YEWed3wY3F)

A golang library for querying the [steam webapi](https://wiki.teamfortress.com/wiki/WebAPI).

## Endpoints

- [x] ISteamApps
    - GetAppList
    - GetServersAtAddress
    - UpToDateCheck

- [x] ISteamEconomy
    - GetAssetClassInfo
    - GetAssetPrices

- [x] ISteamNews
    - GetNewsForApp

- [x] ISteamUser
    - GetFriendList
    - GetPlayerBans
    - GetPlayerSummaries
    - GetUserGroupList
    - ResolveVanityURL

- [x] IPlayerService
    - GetRecentlyPlayedGames
    - GetOwnedGames
    - GetSteamLevel
    - GetBadges
    - GetCommunityBadgeProgress
    
- [x] ISteamWebAPIUtil
    - GetServerInfo
    - GetSupportedAPIList

- [ ] IEconItems_<AppID>
    - [x] GetPlayerItems
    - [x] GetSchema
    - [x] GetSchemaURL
    - [x] GetStoreMetadata
    - [ ] GetStoreStatus

- [x] Extra Non-WebAPIs functions
  - [x] GetGroupMembers - Return a list of steamids belonging to a steam group

## Example Usage
```go
package main

import (
  "context"
  "fmt"
  "github.com/leighmacdonald/steamid/v3/steamid"
  "github.com/leighmacdonald/steamweb/v2"
  "os"
)

func main() {
    // The env var STEAM_TOKEN can also be used to set the key instead of 
    // calling SetKey directly.
    if err := steamweb.SetKey("XXXXXXXXXXXXXXXXXXXXX"); err != nil {
        fmt.Printf("Error setting steam key: %v", err)  
        os.Exit(1)
    }
    ids := steamid.Collection{steamid.New(76561198132612090), steamid.New(76561197960435530)}
    summaries, _ := steamweb.PlayerSummaries(context.Background(), ids)
    for _, summary := range summaries {
        fmt.Println(summary)        
    }
}
```
