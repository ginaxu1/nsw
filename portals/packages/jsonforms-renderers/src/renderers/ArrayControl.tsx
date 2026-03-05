import { createDefaultValue, type ArrayControlProps, type JsonSchema } from '@jsonforms/core';
import { withJsonFormsArrayControlProps, JsonFormsDispatch } from '@jsonforms/react';
import { Card, Button, Flex, Text, Box } from '@radix-ui/themes';
import { PlusIcon, TrashIcon } from '@radix-ui/react-icons';

export const ArrayControl = ({
    data,
    path,
    schema,
    uischema,
    enabled,
    visible,
    addItem,
    removeItems,
    rootSchema,
    arraySchema,
}: ArrayControlProps) => {
    // If `arraySchema` is present, `schema` is already our `itemsSchema`, else fall back to `schema.items`
    const itemsSchema = arraySchema ? schema : schema.items;
    const actualArraySchema = arraySchema || schema;

    if (visible === false) {
        return null;
    }
    if (!itemsSchema || typeof itemsSchema !== 'object' || Array.isArray(itemsSchema)) {
        return null;
    }

    // After the guard, we know itemsSchema is a valid single JsonSchema object
    const validItemsSchema = itemsSchema as JsonSchema;

    const items = Array.isArray(data) ? data : [];
    const title = actualArraySchema.title || 'Array Items';

    const handleAddItem = () => {
        const newItem = createDefaultValue(validItemsSchema, rootSchema);
        if (addItem) addItem(path, newItem)();
    };

    const handleRemoveItem = (indexToRemove: number) => {
        if (removeItems) {
            const removeFunc = removeItems(path, [indexToRemove]);
            if (removeFunc) removeFunc();
        }
    };

    return (
        <Box mb="6">
            <Flex direction="column" gap="4">
                <Text as="div" size="4" weight="bold">
                    {title}
                </Text>

                {items.length === 0 && (
                    <Box py="4" px="4" style={{ backgroundColor: 'var(--gray-3)', borderRadius: 'var(--radius-3)' }}>
                        <Text size="2" color="gray">No items have been added yet.</Text>
                    </Box>
                )}

                {items.map((_item, index) => {
                    const childPath = `${path}.${index}`;
                    return (
                        <Card key={childPath} size="3" variant="surface">
                            <Flex direction="column" gap="4">
                                <Flex justify="between" align="center">
                                    <Text size="3" weight="bold">
                                        Item {index + 1}
                                    </Text>
                                    {enabled && (
                                        <Button
                                            type="button"
                                            color="red"
                                            variant="soft"
                                            onClick={() => handleRemoveItem(index)}
                                            title="Remove item"
                                        >
                                            <TrashIcon />
                                            Remove
                                        </Button>
                                    )}
                                </Flex>

                                <Box>
                                    <JsonFormsDispatch
                                        schema={validItemsSchema}
                                        uischema={
                                            uischema.options?.detail || (
                                                validItemsSchema.type === 'object' || validItemsSchema.properties ? {
                                                    type: 'VerticalLayout',
                                                    elements: Object.keys(
                                                        validItemsSchema.properties || {}
                                                    ).map((key) => ({
                                                        type: 'Control',
                                                        scope: `#/properties/${key}`,
                                                    })),
                                                } : {
                                                    type: 'Control',
                                                    scope: '#',
                                                }
                                            )
                                        }
                                        path={childPath}
                                        enabled={enabled}
                                        renderers={undefined} /* use inherited renderers */
                                        cells={undefined} /* use inherited cells */
                                    />
                                </Box>
                            </Flex>
                        </Card>
                    );
                })}

                {enabled && (
                    <Box mt="2">
                        <Button
                            type="button"
                            variant="surface"
                            onClick={handleAddItem}
                        >
                            <PlusIcon />
                            Add Item
                        </Button>
                    </Box>
                )}
            </Flex>
        </Box>
    );
};

export default withJsonFormsArrayControlProps(ArrayControl);
