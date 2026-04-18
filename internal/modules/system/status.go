package system

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ServeStatusPage serves the animated root landing page for the backend
func ServeStatusPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	
	// Default status is running unless something critical fails.
	// We can enhance this later to check DB connectivity.
	statusText := "BandhanNova Mind Running"
	statusColor := "#00ff88"
	
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BandhanNova Backend</title>
    <link href="https://fonts.googleapis.com/css2?family=Orbitron:wght@700;900&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #050a15;
            --text-color: STATUS_COLOR;
            --glow-color: STATUS_COLOR;
        }
        
        body, html {
            margin: 0;
            padding: 0;
            width: 100%;
            height: 100%;
            background-color: var(--bg-color);
            display: flex;
            justify-content: center;
            align-items: center;
            font-family: 'Orbitron', sans-serif;
            overflow: hidden;
        }

        .container {
            text-align: center;
            position: relative;
        }

        .typo {
            font-size: 5vw;
            font-weight: 900;
            color: transparent;
            -webkit-text-stroke: 2px var(--text-color);
            text-transform: uppercase;
            letter-spacing: 5px;
            position: relative;
            animation: pulse-glow 2s infinite alternate;
        }

        .typo::before {
            content: "STATUS_TEXT";
            position: absolute;
            left: 0;
            top: 0;
            width: 0%;
            height: 100%;
            color: var(--text-color);
            -webkit-text-stroke: 0px transparent;
            overflow: hidden;
            border-right: 4px solid var(--text-color);
            animation: typing 4s steps(30) infinite alternate, cursor-blink 0.5s step-end infinite alternate;
            filter: drop-shadow(0 0 15px var(--glow-color));
        }

        .dots {
            display: inline-block;
            min-width: 30px;
            text-align: left;
            animation: dots-anim 2s infinite steps(4);
        }

        @keyframes pulse-glow {
            0% { text-shadow: 0 0 10px rgba(0,255,136,0.1); }
            100% { text-shadow: 0 0 30px rgba(0,255,136,0.4); }
        }

        @keyframes typing {
            0% { width: 0; }
            100% { width: 100%; }
        }

        @keyframes cursor-blink {
            50% { border-color: transparent; }
        }

        @keyframes dots-anim {
            0% { content: ""; }
            25% { content: "."; }
            50% { content: ".."; }
            75% { content: "..."; }
            100% { content: "..."; }
        }

        .ambient-light {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            width: 50vw;
            height: 50vw;
            background: radial-gradient(circle, rgba(0,255,136,0.05) 0%, rgba(0,0,0,0) 70%);
            z-index: -1;
            pointer-events: none;
            animation: breath 6s infinite ease-in-out;
        }

        @keyframes breath {
            0%, 100% { transform: translate(-50%, -50%) scale(1); opacity: 0.5; }
            50% { transform: translate(-50%, -50%) scale(1.2); opacity: 1; }
        }
    </style>
</head>
<body>
    <div class="ambient-light"></div>
    <div class="container">
        <div class="typo">STATUS_TEXT<span class="dots">...</span></div>
    </div>
</body>
</html>
	`
	
	html = strings.ReplaceAll(html, "STATUS_TEXT", statusText)
	html = strings.ReplaceAll(html, "STATUS_COLOR", statusColor)
	
	return c.SendString(html)
}
