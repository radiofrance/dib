import { sveltekit } from '@sveltejs/kit/vite';
import { resolve } from 'path';

/** @type {import('vite').UserConfig} */
const config = {
	plugins: [sveltekit()],
	resolve: {
		alias: {
			$models: resolve('src/models'),
			$stores: resolve('src/stores')
		}
	}
};

export default config;
