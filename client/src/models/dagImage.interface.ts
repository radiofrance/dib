import type { GossResults } from '$models/goss.interface';
import type { TrivyResults } from '$models/trivy.interface';

export interface Image {
	name: string;
	docker: string;
	goss: GossResults;
	trivy: TrivyResults;
}
