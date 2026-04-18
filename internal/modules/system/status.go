package system

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)

// ServeStatusPage serves the animated root landing page for the backend
func ServeStatusPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BandhanNova Mind | Running</title>
    <style>
        :root {
            --bg: #030712;
            --neon-blue: #00d2ff;
            --neon-purple: #9d50bb;
            --text: #e5e7eb;
        }

        body {
            margin: 0;
            padding: 0;
            background: var(--bg);
            color: var(--text);
            font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
            height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            overflow: hidden;
        }

        .container {
            text-align: center;
            position: relative;
        }

        /* Rounded Loading Object */
        .orb-container {
            position: relative;
            width: 200px;
            height: 200px;
            margin-bottom: 40px;
        }

        .orb {
            position: absolute;
            width: 100%;
            height: 100%;
            border-radius: 50%;
            border: 2px solid transparent;
            border-top: 2px solid var(--neon-blue);
            border-bottom: 2px solid var(--neon-purple);
            animation: spin 3s linear infinite;
        }

        .orb-inner {
            position: absolute;
            top: 20px; left: 20px; right: 20px; bottom: 20px;
            border-radius: 50%;
            background: radial-gradient(circle, rgba(0,210,255,0.1) 0%, rgba(157,80,187,0.05) 100%);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255,255,255,0.1);
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 0 50px rgba(0,210,255,0.2);
        }

        .orb-core {
            width: 40px;
            height: 40px;
            background: var(--text);
            border-radius: 50%;
            box-shadow: 0 0 30px #fff, 0 0 60px var(--neon-blue);
            animation: pulse 2s infinite ease-in-out;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        @keyframes pulse {
            0%, 100% { transform: scale(1); opacity: 0.8; }
            50% { transform: scale(1.2); opacity: 1; }
        }

        h1 {
            font-size: 2.5rem;
            font-weight: 300;
            letter-spacing: 5px;
            margin: 0;
            text-transform: uppercase;
            background: linear-gradient(90deg, var(--neon-blue), var(--neon-purple));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            text-shadow: 0 10px 20px rgba(0,0,0,0.5);
        }

        .status-text {
            margin-top: 10px;
            font-size: 1.1rem;
            color: rgba(255,255,255,0.5);
            letter-spacing: 2px;
        }

        /* Animated Dots */
        .dots::after {
            content: '';
            animation: dots 1.5s steps(4, end) infinite;
        }

        @keyframes dots {
            0%, 20% { content: ''; }
            40% { content: '.'; }
            60% { content: '..'; }
            80%, 100% { content: '...'; }
        }

        .badge {
            margin-top: 30px;
            padding: 8px 16px;
            background: rgba(255,255,255,0.05);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 20px;
            font-size: 0.8rem;
            color: var(--neon-blue);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="orb-container">
            <div class="orb"></div>
            <div class="orb-inner">
                <div class="orb-core"></div>
            </div>
        </div>
        <h1>BandhanNova Mind</h1>
        <div class="status-text">RUNNING<span class="dots"></span></div>
        <div class="badge">BFOBS GATEWAY v2.1</div>
    </div>
</body>
</html>
	`
	return c.SendString(html)
}

// FormatLatency is a helper function (not used in HTML but keep for utility)
func FormatLatency(l int64) string {
	return fmt.Sprintf("%dms", l)
}
