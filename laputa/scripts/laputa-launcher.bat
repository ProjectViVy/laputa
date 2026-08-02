@echo off
REM Laputa launcher wrapper for use in hermes shell hooks.
REM Invokes laputa-launcher.py via the hermes-agent venv's Python
REM (Windows' bare 'python' resolves to the Microsoft Store stub
REM and prints "Python was not found" — see Windows App Execution Aliases).
"C:\Users\Administrator\AppData\Local\hermes\hermes-agent\venv\Scripts\python.exe" "C:\Users\Administrator\Desktop\projects\laputa\scripts\laputa-launcher.py" %*