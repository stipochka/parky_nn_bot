from telethon.sync import TelegramClient
import sys
from dotenv import load_dotenv
import json
import os


load_dotenv()
api_id   = int(os.getenv("API_ID"))
api_hash = os.getenv("API_HASH")
group    = os.getenv("GROUP")

query = sys.argv[1]
if len(query) > 2:
    query = query[:-2]
results = []
session_path = "/app/session.session"
with TelegramClient(session_path, api_id, api_hash) as client:
    entity = client.get_entity(group)

    for message in client.iter_messages(group, limit=100):
        if message.text and query.lower() in message.text.lower():
            if entity.username:
                link = f"https://t.me/{entity.username}/{message.id}"
            else:
                link = f"https://t.me/c/{entity.id // 10**9}/{message.id}"  # для приватных
            results.append({
                "date": message.date.isoformat(),
                "link": link
            })
        if len(results) >= 5:
            break

results.sort(key=lambda x: x["date"], reverse=True)

print(json.dumps(results, indent=2))

