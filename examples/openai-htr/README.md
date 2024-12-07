# openai-htr

Use OpenAI ChatGPT to transcribe handwritten text.

## Secrets

Requires an environment variable `OPENAI_API_KEY`

If [deploying this in kubernetes](../../ci/k8s/htr.yaml), you can create the secret via

```
 kubectl create secret generic openai \
  --from-literal=api-key=$OPENAI_API_KEY
```
