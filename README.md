# Codebase Chat with LLM  

*Early Development Stage*  

A Golang-powered service that lets you chat with an LLM (via Ollama) to get insights about GitHub repositories, commit histories, and code changes. Ask questions like:  
- "Are any commits using a specific method?"  
- "Who modified this file recently?"  
- "Explain the diff in commit XYZ."  

---

## Features  
- **Codebase Q&A**: Query commits, diffs, and metadata using natural language.  
- **Ollama Integration**: Supports both embedding and conversational Ollama models.  
- **ChromaDB Vector Store**: Stores and retrieves commit embeddings for fast similarity search.  
- **GitHub Integration**: Fetches repository data using a GitHub access token.  

**Work in Progress**  
- Conversational memory (refer to previous messages in prompts)  
- Customizable ChromaDB URL/port  
- Improved CLI tooling
- Frontend
- etc.

---

## Requirements  
1. **ChromaDB Server** running at `http://localhost:8000` (default)  
2. **Ollama Instance** with:  
   - An embedding model (e.g., `nomic-embed-text`)  
   - A conversational model (e.g., `llama3`, `mistral`)  
3. **GitHub Access Token** (for querying repositories)  

---

