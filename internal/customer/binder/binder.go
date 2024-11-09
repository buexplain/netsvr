/**
* Copyright 2023 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

// Package binder 网关唯一id与客户业务系统唯一id的绑定关系
package binder

import (
	"sync"
)

// binder 客户id与连接id的映射关系
type binder struct {
	//一个客户id有多个连接id
	customerIdToUniqIds map[string]map[string]struct{}
	//一个连接id只允许有一个客户id
	uniqIdToCustomerId map[string]string
	rwMutex            *sync.RWMutex
}

var Binder *binder

func init() {
	Binder = &binder{
		customerIdToUniqIds: make(map[string]map[string]struct{}),
		uniqIdToCustomerId:  make(map[string]string),
		rwMutex:             &sync.RWMutex{},
	}
}

// GetCustomerIds 获取所有的customerId
func (r *binder) GetCustomerIds() (customerIds []string) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	customerIds = make([]string, 0, len(r.customerIdToUniqIds))
	for customerId := range r.customerIdToUniqIds {
		customerIds = append(customerIds, customerId)
	}
	return customerIds
}

// GetUniqIdsByCustomerId 根据customerId获取所有的uniqId
func (r *binder) GetUniqIdsByCustomerId(customerId string) (uniqIds []string) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	if uniqIdsMap, ok := r.customerIdToUniqIds[customerId]; ok {
		uniqIds = make([]string, 0, len(uniqIdsMap))
		for uniqId := range uniqIdsMap {
			uniqIds = append(uniqIds, uniqId)
		}
		return uniqIds
	}
	return nil
}

// GetUniqIdsByCustomerIds 根据customerIds获取所有的uniqId
func (r *binder) GetUniqIdsByCustomerIds(customerIds []string) (customerIdUniqIds map[string][]string) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	customerIdUniqIds = make(map[string][]string, len(customerIds))
	for _, customerId := range customerIds {
		uniqIdsMap, ok := r.customerIdToUniqIds[customerId]
		if !ok {
			continue
		}
		uniqIds := make([]string, 0, len(uniqIdsMap))
		for uniqId := range uniqIdsMap {
			uniqIds = append(uniqIds, uniqId)
		}
		customerIdUniqIds[customerId] = uniqIds
	}
	return customerIdUniqIds
}

// GetCustomerIdsByUniqIds 根据uniqIds获取所有的customerId
func (r *binder) GetCustomerIdsByUniqIds(uniqIds []string) (customerIds []string) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	//去重
	customerIdsMap := make(map[string]struct{}, len(uniqIds))
	for _, uniqId := range uniqIds {
		if customerId, ok := r.uniqIdToCustomerId[uniqId]; ok {
			customerIdsMap[customerId] = struct{}{}
		}
	}
	//转换
	customerIds = make([]string, 0, len(customerIdsMap))
	for customerId := range customerIdsMap {
		customerIds = append(customerIds, customerId)
	}
	return customerIds
}

// GetCustomerIdToUniqIdsList 根据uniqIds获取所有的customerId对应的uniqIds
func (r *binder) GetCustomerIdToUniqIdsList(uniqIds []string) (customerIdToUniqIdsList map[string][]string) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	//去重
	customerIdToUniqIdsList = make(map[string][]string, len(uniqIds))
	for _, uniqId := range uniqIds {
		if customerId, ok := r.uniqIdToCustomerId[uniqId]; ok {
			if _, ok := customerIdToUniqIdsList[customerId]; ok {
				customerIdToUniqIdsList[customerId] = append(customerIdToUniqIdsList[customerId], uniqId)
			} else {
				customerIdToUniqIdsList[customerId] = []string{uniqId}
			}
		}
	}
	return customerIdToUniqIdsList
}

// CountCustomerIdsByUniqIds 根据uniqIds获取所有的customerId数量
func (r *binder) CountCustomerIdsByUniqIds(uniqIds []string) int {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	//去重
	customerIdsMap := make(map[string]struct{}, len(uniqIds))
	for _, uniqId := range uniqIds {
		if customerId, ok := r.uniqIdToCustomerId[uniqId]; ok {
			customerIdsMap[customerId] = struct{}{}
		}
	}
	return len(customerIdsMap)
}

func (r *binder) Len() int {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	return len(r.customerIdToUniqIds)
}

func (r *binder) Set(uniqId string, customerId string) {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()
	if currentCustomerId, ok := r.uniqIdToCustomerId[uniqId]; ok {
		//如果当前uniqId已经绑定了customerId，并且新的customerId和旧的customerId相同，则不处理
		if currentCustomerId == customerId {
			return
		}
		//如果当前uniqId已经绑定了customerId，并且新的customerId和旧的customerId不相同，则删除旧的customerId的uniqId
		if uniqIdsMap, ok := r.customerIdToUniqIds[currentCustomerId]; ok {
			delete(uniqIdsMap, uniqId)
			if len(uniqIdsMap) == 0 {
				delete(r.customerIdToUniqIds, currentCustomerId)
			}
		}
	}
	//设置新的
	r.uniqIdToCustomerId[uniqId] = customerId
	if uniqIds, ok := r.customerIdToUniqIds[customerId]; ok {
		uniqIds[uniqId] = struct{}{}
	} else {
		r.customerIdToUniqIds[customerId] = map[string]struct{}{uniqId: {}}
	}
}

func (r *binder) DelUniqId(uniqId string) {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()
	if customerId, ok := r.uniqIdToCustomerId[uniqId]; ok {
		delete(r.uniqIdToCustomerId, uniqId)
		uniqIdsMap := r.customerIdToUniqIds[customerId]
		delete(uniqIdsMap, uniqId)
		if len(uniqIdsMap) == 0 {
			delete(r.customerIdToUniqIds, customerId)
		}
	}
}
