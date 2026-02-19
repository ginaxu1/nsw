import { vanillaRenderers } from '@jsonforms/vanilla-renderers';
import { FileControl, FileControlTester } from '@opennsw/ui';

export { FileControl, FileControlTester };
export const customRenderers = [
    ...vanillaRenderers,
    { tester: FileControlTester, renderer: FileControl },
];
