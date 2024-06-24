import { sveltekit } from '@sveltejs/kit/vite';
import { viteStaticCopy } from 'vite-plugin-static-copy';
import { resolve } from 'path';

// Copy report data fixtures only during development
const viteStaticCopyReportData = viteStaticCopy({
	targets: [
		{
			src: resolve('./fixtures/report-data/dev/*'),
			dest: './'
		}
	]
});

/** @type {import('vite').UserConfig} */
const config = {
	plugins: [sveltekit(), process.env.NODE_ENV !== 'production' ? viteStaticCopyReportData : null],
	resolve: {
		alias: {
			$models: resolve('src/models'),
			$stores: resolve('src/stores')
		}
	}
};

export default config;
