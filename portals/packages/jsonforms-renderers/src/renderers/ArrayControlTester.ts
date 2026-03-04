import { rankWith, isObjectArrayControl } from '@jsonforms/core';

export const ArrayControlTester = rankWith(
    3,
    isObjectArrayControl
);
