# steamweb

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Test, Build & Publish](https://github.com/leighmacdonald/steamweb/actions/workflows/build.yml/badge.svg?branch=master)](https://github.com/leighmacdonald/steamweb/actions/workflows/build.yml)
[![release](https://github.com/leighmacdonald/steamweb/actions/workflows/release.yml/badge.svg?event=release)](https://github.com/leighmacdonald/steamweb/actions/workflows/release.yml)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f06234b0551a49cc8ac111d7b77827b2)](https://www.codacy.com/manual/leighmacdonald/steamweb?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=leighmacdonald/steamweb&amp;utm_campaign=Badge_Grade)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmacdonald/steamweb)](https://goreportcard.com/report/github.com/leighmacdonald/steamweb)
[![GoDoc](https://godoc.org/github.com/leighmacdonald/steamweb?status.svg)](https://pkg.go.dev/github.com/leighmacdonald/steamweb)
[![Discord chat](https://img.shields.io/discord/704508824320475218)](https://discord.gg/YEWed3wY3F)

A golang library for querying the [steam webapi](https://wiki.teamfortress.com/wiki/WebAPI).

## Endpoints

- [x] ISteamApps
    - [x] GetAppList
    - [x] GetServersAtAddress
    - [x] UpToDateCheck

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
    
## Example Usage

    import (
        "fmt"
        "github.com/leighmacdonald/steamweb"
        "os"
    )

    func main() {
        // The env var STEAM_TOKEN can also be used to set the key instead of 
        // calling SetKey directly.
        if err := steamweb.SetKey("XXXXXXXXXXXXXXXXXXXXX"); err != nil {
            fmt.Printf("Error setting steam key: %v", err)  
            os.Exit(1)
        }
        ids := steamid.Collection{76561198132612090, testIDSquirrelly, 76561197960435530}
	    summaries, _ := steamweb.PlayerSummaries(ids)
        for _, summary := range summaries {
            fmt.Println(summary)        
        }
    }

