#!/bin/bash

# =============================================================================
# KubeSynapse Doc Ingestion - MD to Qdrant Vector DB
# =============================================================================

FILE_PATH=$1

if [ -z "$FILE_PATH" ]; then
    echo "Usage: ./ingest_docs.sh path/to/document.md"
    exit 1
fi

if [ ! -f "$FILE_PATH" ]; then
    echo "Error: File not found: $FILE_PATH"
    exit 1
fi

CONTENT=$(cat "$FILE_PATH" | tr '\n' ' ' | sed 's/"/\\"/g')

echo "📖 Reading: $FILE_PATH"

# 1. Get Embedding from Ollama (using the nomic model we pulled earlier)
echo "⌛ Generating Embedding from Ollama..."
EMBEDDING=$(curl -s http://localhost:31434/api/embeddings \
-d "{
  \"model\": \"nomic-embed-text\",
  \"prompt\": \"$CONTENT\"
}" | jq -c '.embedding')

if [ -z "$EMBEDDING" ] || [ "$EMBEDDING" == "null" ]; then
    echo "❌ Error: Failed to generate embedding. Is Ollama running at localhost:31434?"
    exit 1
fi

# 2. Upload to Qdrant
echo "🚀 Uploading Point to Qdrant (k8s-incidents)..."
POINT_ID=$(date +%s)
curl -s -X PUT "http://localhost:30633/collections/k8s-incidents/points" \
     -H "Content-Type: application/json" \
     --data "{
        \"points\": [
            {
                \"id\": $POINT_ID,
                \"vector\": $EMBEDDING,
                \"payload\": {
                    \"source\": \"$FILE_PATH\",
                    \"content\": \"$CONTENT\",
                    \"timestamp\": \"$(date)\"
                }
            }
        ]
    }" | jq .

echo "✅ Knowledge Ingested Successfully!"
