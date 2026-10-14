$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Speech
$taskRoot = Split-Path -Parent $PSScriptRoot
$taskAudioDirectory = Join-Path $taskRoot '.runtime/demo-audio'
New-Item -ItemType Directory -Path $taskAudioDirectory -Force | Out-Null
$taskNarration = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'demo-narration.json') -Raw -Encoding UTF8 | ConvertFrom-Json
for ($taskIndex=0; $taskIndex -lt $taskNarration.Count; $taskIndex++) {
    $taskSynth = New-Object System.Speech.Synthesis.SpeechSynthesizer
    try {
        $taskSynth.SelectVoice('Microsoft Zira Desktop')
        $taskSynth.Rate = 0
        $taskSynth.SetOutputToWaveFile((Join-Path $taskAudioDirectory ("$taskIndex.wav")))
        $taskSynth.Speak($taskNarration[$taskIndex].text)
    } finally { $taskSynth.Dispose() }
}
Write-Host 'Offline English narration generated; no speaker playback or external service.'
