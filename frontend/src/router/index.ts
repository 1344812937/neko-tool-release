import {createRouter, createWebHistory} from 'vue-router'
import { AUTH_ROUTE_PATH, getStoredAuthKey } from '@/utils/auth'

const routes = [
    { path: '/auth', component: () => import('@/pages/Auth.vue') },
    { path: '/', component: () => import('@/pages/Home.vue') },
    { path: '/site/logs', component: () => import('@/pages/SiteLogs.vue') },
    { path: '/projects', component: () => import('@/pages/Projects.vue') },
    { path: '/projects/:id/browser', component: () => import('@/pages/ProjectBrowser.vue') },
    { path: '/nodes', component: () => import('@/pages/Nodes.vue') },
    { path: '/compare', component: () => import('@/pages/Compare.vue') },
    { path: '/demo', component: () => import('@/pages/Demo.vue') },
    { path: '/test', component: () => import('@/pages/Page2.vue') },
]

const router = createRouter({
    history: createWebHistory("/static/"),
    routes,
})

router.beforeEach((to) => {
    if (to.path === AUTH_ROUTE_PATH) {
        return true
    }
    if (!getStoredAuthKey()) {
        return {
            path: AUTH_ROUTE_PATH,
            query: { redirect: to.fullPath },
        }
    }
    return true
})

export default router
