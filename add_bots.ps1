# Add 10 Bots to a room
# Usage: ./add_bots.ps1 <RoomID>

$RoomID = $args[0]
if (-not $RoomID) {
    Write-Host "Usage: ./add_bots.ps1 <RoomID>"
    exit 1
}

$BaseURL = "http://localhost:8080/v1/rooms/$RoomID/join"

Write-Host "Adding 10 bots to room $RoomID..."

for ($i=1; $i -le 10; $i++) {
    $BotName = "Bot_$i"
    $Body = @{
        name = $BotName
    } | ConvertTo-Json
    
    # We need a unique token or player_id for each bot
    # Since endpoint is protected, we need to handle auth. 
    # But wait, looking at backend, joinRoom uses authMiddleware.
    # So we need to Register/Login each bot first.
    
    # Actually, let's use the WebSocket guest login trick or just register them.
    # Easier: Use the "guest" login flow by just generating a token?
    # No, let's just register them via API.
    
    $User = "bot_user_$i"
    $Pass = "password"
    
    # 1. Register
    try {
        $RegBody = @{
            email = "$User@bot.local"
            password = $Pass
        } | ConvertTo-Json
        Invoke-RestMethod -Uri "http://localhost:8080/v1/auth/register" -Method Post -Body $RegBody -ContentType "application/json" -ErrorAction SilentlyContinue
    } catch {}

    # 2. Login
    $LoginBody = @{
        email = "$User@bot.local"
        password = $Pass
    } | ConvertTo-Json
    
    try {
        $LoginResp = Invoke-RestMethod -Uri "http://localhost:8080/v1/auth/login" -Method Post -Body $LoginBody -ContentType "application/json"
        $Token = $LoginResp.token
        
        # 3. Join Room
        $Headers = @{
            Authorization = "Bearer $Token"
        }
        Invoke-RestMethod -Uri $BaseURL -Method Post -Body $Body -ContentType "application/json" -Headers $Headers
        
        Write-Host "Added $BotName"
    } catch {
        Write-Host "Failed to add $BotName : $_"
    }
}

Write-Host "Done."
