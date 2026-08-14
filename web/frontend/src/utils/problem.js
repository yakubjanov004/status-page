/**
 * Loyihalar ro'yxatidan eng birinchi (eng muhim) muammoni topadi —
 * shunda banner va UI aniq "aynan shu joyda muammo bor" deb ko'rsata oladi,
 * o'nlab yashil qatorlar orasida adashib qolmaydi.
 *
 * Ustuvorlik: to'liq ishlamayotgan (down) komponent > qisman uzilish (warn).
 */
export function findPrimaryProblem(projects) {
    if (!projects || projects.length === 0) return null;

    for (const proj of projects) {
        for (const comp of proj.components || []) {
            if (!comp.is_up) {
                return {
                    severity: 'down',
                    projectName: proj.name,
                    projectSlug: proj.slug,
                    componentName: comp.name,
                };
            }
        }
    }

    return null;
}
