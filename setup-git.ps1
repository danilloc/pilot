Write-Host "=== PILOT - Git Setup ===" -ForegroundColor Cyan

if (-not (Test-Path '.git')) {
    git init
    Write-Host "Git initialized" -ForegroundColor Green
}

git config --global user.name 'Danillo'
git config --global user.email 'danillo@example.com'

git branch develop 2>$null
git checkout develop 2>$null

git add .
git commit -m 'chore(initial): setup pilot project structure'

Write-Host 'Setup done! Now add your GitHub remote:' -ForegroundColor Yellow
Write-Host '  git remote add origin https://github.com/seu-usuario/pilot.git' -ForegroundColor White
Write-Host '  git push -u origin develop' -ForegroundColor White