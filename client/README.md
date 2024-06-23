# Dib client

Dib client is a Svelte app that we compile as static-site to embed in final Dib Go binary.

It is used as base to generate "dib build" html report outputs.

see `client.go` if you are curious.

## Developing

```bash
npm install 
npm run dev
```

## Building

To create a production version of your app:

```bash
npm run build
```

You can preview the production build with `npm run preview`.
